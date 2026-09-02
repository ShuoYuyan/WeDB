package storage

// 加密静态存储（encryption at rest）
//
// 设计：
//   - 算法：AES-256-XTS（扇区级可调加密）。密文与明文等长，因此不破坏
//     Pager 的固定页偏移数学；页内完整性由既有 PageHeader.Checksum 保证。
//   - 数据单元 = 一页（默认 4096 字节，必须是 16 的倍数）。
//   - 密钥：PBKDF2-HMAC-SHA256(passphrase, salt, iters, 64) 拆成
//     前 32 字节数据密钥 + 后 32 字节 tweak 密钥。
//   - 验证：<db>.xkey 存 {salt, nonce, verifier}，verifier 为用派生密钥
//     对魔数串的 AES-GCM 密文 —— 口令错误在打开时立即失败。
//
// 仅 stdlib，兼容 Go1.10 镜像。

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

func isNotExistStd(err error) bool { return os.IsNotExist(err) }

const (
	xtsBlock       = 16 // AES 块大小
	xkeySuffix     = ".xkey"
	xkeyMagicPlain = "WeDB-XKEY-v1"
	kdfIterations  = 100000
	kdfSaltLen     = 32
	kdfKeyLen      = 64 // 32 data + 32 tweak
)

// ErrWrongPassphrase 口令错误或未提供
var ErrWrongPassphrase = errors.New("wedb: wrong or missing passphrase")

// sectorCipher 对定长扇区做同尺寸加解密
type sectorCipher interface {
	// EncryptSector 原地加密一个扇区；sectorNo 为逻辑扇区号
	EncryptSector(sectorNo uint64, buf []byte) error
	DecryptSector(sectorNo uint64, buf []byte) error
	SectorSize() int
}

// ------------------------------------------------------------------ XTS

type xtsCrypter struct {
	enc      cipher.Block // 数据密钥
	dec      cipher.Block
	tweakKey [32]byte
	sector   int
}

func newXtsCrypter(key64 []byte, sectorSize int) (*xtsCrypter, error) {
	if len(key64) != 64 {
		return nil, fmt.Errorf("xts: need 64-byte key, got %d", len(key64))
	}
	if sectorSize%xtsBlock != 0 || sectorSize == 0 {
		return nil, fmt.Errorf("xts: sector size %d must be positive multiple of %d", sectorSize, xtsBlock)
	}
	enc, err := aes.NewCipher(key64[:32])
	if err != nil {
		return nil, err
	}
	dec, err := aes.NewCipher(key64[:32])
	if err != nil {
		return nil, err
	}
	c := &xtsCrypter{enc: enc, dec: dec, sector: sectorSize}
	copy(c.tweakKey[:], key64[32:])
	return c, nil
}

func (c *xtsCrypter) SectorSize() int { return c.sector }

// tweakValue 计算扇区号对应的初始 tweak（16 字节大端计数，再被 tweakKey 加密）
func (c *xtsCrypter) tweakValue(sectorNo uint64) [16]byte {
	var t [16]byte
	binary.BigEndian.PutUint64(t[8:], sectorNo)
	c.enc.Encrypt(t[:], t[:])
	return t
}

// gfMul128 在 GF(2^128) 上乘 α（x^128+x^7+x^6+x 约简多项式 0x87）
func gfMul128(t *[16]byte) {
	var carry byte
	for i := 15; i >= 0; i-- {
		nc := t[i] >> 7
		t[i] = t[i]<<1 | carry
		carry = nc
	}
	if carry != 0 {
		t[15] ^= 0x87
	}
}

func (c *xtsCrypter) cryptSector(sectorNo uint64, buf []byte, encrypt bool) error {
	if len(buf)%xtsBlock != 0 || len(buf) == 0 {
		return fmt.Errorf("xts: buffer %d not multiple of %d", len(buf), xtsBlock)
	}
	t := c.tweakValue(sectorNo)

	var pp, cc [16]byte
	for off := 0; off < len(buf); off += xtsBlock {
		block := buf[off : off+xtsBlock]
		for i := 0; i < xtsBlock; i++ {
			pp[i] = block[i] ^ t[i]
		}
		if encrypt {
			c.enc.Encrypt(cc[:], pp[:])
		} else {
			c.dec.Decrypt(cc[:], pp[:])
		}
		for i := 0; i < xtsBlock; i++ {
			cc[i] ^= t[i]
			block[i] = cc[i]
		}
		gfMul128(&t)
	}
	return nil
}

func (c *xtsCrypter) EncryptSector(sectorNo uint64, buf []byte) error {
	return c.cryptSector(sectorNo, buf, true)
}

func (c *xtsCrypter) DecryptSector(sectorNo uint64, buf []byte) error {
	return c.cryptSector(sectorNo, buf, false)
}

// ------------------------------------------------------------------ KDF

// pbkdf2SHA256Std PBKDF2-HMAC-SHA256（RFC 2898），仅 stdlib。
func pbkdf2SHA256Std(password, salt []byte, iter, keyLen int) []byte {
	prf := hmac.New(sha256.New, password)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen

	var buf [4]byte
	dk := make([]byte, 0, numBlocks*hashLen)
	u := make([]byte, hashLen)
	for block := 1; block <= numBlocks; block++ {
		prf.Reset()
		prf.Write(salt)
		buf[0] = byte(block >> 24)
		buf[1] = byte(block >> 16)
		buf[2] = byte(block >> 8)
		buf[3] = byte(block)
		prf.Write(buf[:])
		dk = prf.Sum(dk)
		t := dk[len(dk)-hashLen:]
		copy(u, t)
		for i := 2; i <= iter; i++ {
			prf.Reset()
			prf.Write(u)
			u = u[:0]
			u = prf.Sum(u)
			for x := range u {
				t[x] ^= u[x]
			}
		}
	}
	return dk[:keyLen]
}

// ------------------------------------------------------------------ xkey

type xkeyFile struct {
	Salt     string `json:"salt"`     // base64
	Nonce    string `json:"nonce"`    // base64 (12)
	Verifier string `json:"verifier"` // base64 (GCM seal of magic)
	Iters    int    `json:"iters"`
}

func xkeyPathFor(dbPath string) string { return dbPath + xkeySuffix }

// loadOrCreateXKey 打开（或首次创建）验证文件并返回派生密钥。
// passphrase 为空且不存在 xkey -> 返回 nil,nil,nil（非加密模式）。
func loadOrCreateXKey(dbPath string, passphrase []byte, randReader io.Reader) (dataKey, tweakKey []byte, err error) {
	path := xkeyPathFor(dbPath)
	raw, rerr := os.ReadFile(path)

	if os_IsNotExist(rerr) {
		if len(passphrase) == 0 {
			return nil, nil, nil // 明文模式
		}
		salt := make([]byte, kdfSaltLen)
		if _, err := io.ReadFull(randReader, salt); err != nil {
			return nil, nil, err
		}
		key := pbkdf2SHA256Std(passphrase, salt, kdfIterations, kdfKeyLen)

		block, err := aes.NewCipher(key[:32])
		if err != nil {
			return nil, nil, err
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return nil, nil, err
		}
		nonce := make([]byte, gcm.NonceSize())
		if _, err := io.ReadFull(randReader, nonce); err != nil {
			return nil, nil, err
		}
		verifier := gcm.Seal(nil, nonce, []byte(xkeyMagicPlain), nil)

		xf := xkeyFile{
			Salt:     base64.StdEncoding.EncodeToString(salt),
			Nonce:    base64.StdEncoding.EncodeToString(nonce),
			Verifier: base64.StdEncoding.EncodeToString(verifier),
			Iters:    kdfIterations,
		}
		if err := writeJSONFile(path, &xf); err != nil {
			return nil, nil, err
		}
		return key[:32], key[32:], nil
	}
	if rerr != nil {
		return nil, nil, rerr
	}
	if len(passphrase) == 0 {
		return nil, nil, ErrWrongPassphrase // 已加密库必须提供口令
	}

	var xf xkeyFile
	if err := unmarshalJSONFile(raw, &xf); err != nil {
		return nil, nil, err
	}
	salt, err := base64.StdEncoding.DecodeString(xf.Salt)
	if err != nil {
		return nil, nil, err
	}
	nonce, err := base64.StdEncoding.DecodeString(xf.Nonce)
	if err != nil {
		return nil, nil, err
	}
	verifier, err := base64.StdEncoding.DecodeString(xf.Verifier)
	if err != nil {
		return nil, nil, err
	}
	key := pbkdf2SHA256Std(passphrase, salt, xf.Iters, kdfKeyLen)

	block, err := aes.NewCipher(key[:32])
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	plain, err := gcm.Open(nil, nonce, verifier, nil)
	if err != nil || string(plain) != xkeyMagicPlain {
		return nil, nil, ErrWrongPassphrase
	}
	return key[:32], key[32:], nil
}

// 小工具：隔离文件与 JSON 读写，便于镜像转换与测试替换
func os_IsNotExist(err error) bool {
	return err != nil && osIsNotExist(err)
}

func osIsNotExist(err error) bool {
	// 直接复用标准库判定
	return isNotExistStd(err)
}

func writeJSONFile(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func unmarshalJSONFile(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
