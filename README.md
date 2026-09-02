# WeDB

> **Embedded document database with ODBC 3.x driver**  
> **嵌入式文档数据库 + ODBC 3.x 驱动**

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8.svg)](https://go.dev/)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey.svg)]()
[![DCO](https://img.shields.io/badge/Contributing-DCO-brightgreen.svg)](CONTRIBUTING.md)

WeDB is a pure-Go embedded document database with a built-in ODBC 3.x driver
for Windows. It is designed for single-process, single-host use cases
(multi-tenant CMS, embedded analytics, desktop applications) and offers
table model, ordered primary keys, unique indexes, cross-table atomic
transactions, MVCC isolation, and B+ tree page persistence.

WeDB 是一个纯 Go 嵌入式文档数据库，自带 Windows ODBC 3.x 驱动。面向单进程
单主机场景（多租户 CMS、嵌入式分析、桌面应用），提供表模型、有序主键、
唯一索引、跨表原子事务、MVCC 隔离与 B+ 树页式持久化。

---

## ✨ Features / 特性

### Database Core / 数据库核心
- **Pure Go** (no cgo in the storage engine) / **纯 Go**（存储引擎无 cgo）
- **B+ tree page persistence** with checksums / **B+ 树页式持久化**，带校验和
- **MVCC isolation** (READ UNCOMMITTED / READ COMMITTED / REPEATABLE READ / SNAPSHOT)
- **Per-table write locks** — different tables write fully in parallel
  (vs. SQLite's library-wide single writer) / **每表一把读写锁**——不同表的写完全并行（对比 SQLite 的库级单写）
- **Atomic cross-table transactions** with serial write gate / **跨表原子事务**，单写闸门串行化
- **AES-256-XTS at-rest encryption** (PBKDF2-HMAC-SHA256 key derivation)
- **File lock** against multi-process corruption / **文件锁**防多进程损坏

### ODBC 3.x Driver (Windows) / ODBC 3.x 驱动（Windows）
- 74 exported functions (ANSI + Unicode W variants + Config*)
- Complete SQL-92 subset: CREATE/DROP TABLE, INSERT, SELECT (WHERE/ORDER BY/LIMIT/OFFSET/aggregates)
- Direct CGO export as `wedb_odbc.dll`
- C, Delphi, and Python clients tested
- Full SQLSTATE diagnostics

---

## 📦 Project Layout / 项目结构

```
WeDB/
├── cmd/wedb/                 # CLI tool (interactive) / 命令行工具
├── internal/
│   ├── api/                  # Public types (TableSchema, IndexInfo, TxOptions)
│   ├── storage/              # Core engine (B-tree / Pager / MVCC / tx_staging)
│   ├── types/                # Schema / Column internal types
│   └── util/                 # Logger, validator, pool
├── pkg/adapter/              # WeDBAdapter facade / 门面
├── drivers/odbc/             # ODBC 3.x driver (CGO → wedb_odbc.dll)
│   ├── api_core.go           # Core SQL entry points (38 ANSI + 30 W)
│   ├── api_meta.go           # SQLTables / SQLColumns / SQLGetTypeInfo
│   ├── api_meta_info.go      # SQLGetInfo constants
│   ├── handle/               # env/dbc/stmt handle pool
│   ├── sql/                  # SQL parser + executor + WHERE evaluator
│   ├── diag/                 # SQLSTATE constants
│   ├── util_string.go        # Go ↔ C string conversion
│   ├── test/                 # C, Delphi, Python test clients
│   ├── README.md             # ODBC driver documentation
│   ├── STATUS.md             # Implementation status log
│   └── RETROSPECTIVE.md      # Engineering retrospective (lessons learned)
├── WQL/                      # WQL query language prototype (separate Go module)
├── tools/                    # Go 1.10 compatibility shims
├── LICENSE                   # Business Source License 1.1
├── NOTICE                    # Copyright notice
├── CONTRIBUTING.md           # How to contribute (DCO)
└── README.md                 # This file
```

---

## 🚀 Quick Start / 快速开始

### Prerequisites / 前置条件
- Go 1.23 or later / Go 1.23 或更高
- GCC toolchain (for ODBC driver build) / GCC 工具链（构建 ODBC 驱动用）
- Windows (for ODBC driver) / Windows（ODBC 驱动）

### Build the Database / 构建数据库
```bash
go build ./...
```

### Run the CLI / 运行 CLI
```bash
go run ./cmd/wedb
```

### Build the ODBC Driver (Windows) / 构建 ODBC 驱动（Windows）
```powershell
cd drivers/odbc
.\build.ps1
```
Output: `drivers/odbc/build/wedb_odbc.dll`

### Use from C / 从 C 调用
```c
#include <windows.h>
HMODULE dll = LoadLibraryA("wedb_odbc.dll");
// See drivers/odbc/test/direct.c for a complete example
```

### Use from Delphi / 从 Delphi 调用
See `drivers/odbc/test/WeDBODBC.dpr` for a complete Delphi 7/XE2 example.

### Use from Python / 从 Python 调用
See `drivers/odbc/test/pyodbc_test.py`.

---

## 📊 Performance Highlights / 性能亮点

| Feature | WeDB | SQLite |
|---|---|---|
| Multi-table parallel writes | ✅ Per-table lock | ❌ Library-wide lock |
| MVCC snapshot isolation | ✅ All 4 levels | ⚠️ Limited |
| At-rest encryption | ✅ AES-256-XTS | ❌ External (SEE) |
| Pure Go (no cgo) | ✅ Core engine | ❌ (cgo for encryption) |

See `docs/` for detailed benchmarks.

---

## 📚 Documentation / 文档

- **ODBC Driver**: [`drivers/odbc/README.md`](drivers/odbc/README.md) |
  [`STATUS.md`](drivers/odbc/STATUS.md) |
  [`RETROSPECTIVE.md`](drivers/odbc/RETROSPECTIVE.md)
- **API Reference**: `internal/api/` (Go doc comments)
- **Architecture**: `internal/storage/` (see `database.go` for the entry point)

---

## 🤝 Contributing / 参与贡献

We welcome contributions! See [`CONTRIBUTING.md`](CONTRIBUTING.md) for
the DCO sign-off process. By contributing, you agree your work will be
licensed under BUSL-1.1.

欢迎贡献！请阅读 [`CONTRIBUTING.md`](CONTRIBUTING.md) 了解 DCO 签署流程。
贡献意味着您同意您的作品以 BUSL-1.1 协议授权。

---

## 📜 License / 许可证

**GNU Affero General Public License v3 (AGPL-3.0)** — see [`LICENSE`](LICENSE).

WeDB is **true open source** (OSI-approved). You may freely use, modify,
and distribute it, subject to the AGPL-3.0 terms.

### What AGPL-3.0 means in practice / AGPL-3.0 实际意义

| ✅ You CAN / 可以 | ❌ You CANNOT / 不可以 |
|---|---|
| Read and study the source code / 阅读和学习源码 | Take the code and offer it as a closed-source service / 拿去代码作为闭源服务提供 |
| Use it for your own projects (including commercial) / 用于自己的项目（包括商业项目） | Modify it and keep modifications proprietary / 修改后保持专有 |
| Fork it and contribute back / Fork 并贡献回来 | Use it in a SaaS product without open-sourcing your changes / 在 SaaS 产品中使用而不开源你的修改 |
| Distribute it (with the same license) / 分发（需保留相同许可） | Sublicense under different terms / 以不同条款再许可 |
| Sell commercial support, consulting, training around it / 围绕它销售商业支持、咨询、培训 | |

### Section 13 — Network Use is Distribution

AGPL's defining feature (Section 13): if you run a modified version of
WeDB as a network service, you **must** offer the source code of your
modifications to the users of that service. This prevents the
"open-core as a service" loophole that MIT/BSD licenses allow.

AGPL 的核心特性（第 13 条）：如果你将修改后的 WeDB 作为网络服务运行，
**必须**向该服务的用户提供你修改后的源代码。这防止了 MIT/BSD 许可证
允许的"开源核心作为服务"漏洞。

### Commercial Licensing / 商业许可

The author retains the right to offer WeDB under **separate commercial
license terms** for organizations that require proprietary use without
AGPL obligations. This is the same model used by MongoDB, MySQL, and
other successful open-source companies.

作者保留以**单独商业许可条款**提供 WeDB 的权利，适用于需要专有使用
（无 AGPL 义务）的组织。这与 MongoDB、MySQL 等成功的开源公司模式相同。

If your organization needs:
- Proprietary use without AGPL obligations
- Indemnification
- SLA / support contracts
- Custom features not in the open-source edition

Please contact: **https://github.com/ShuoYuyan**

如需以下服务，请联系：**https://github.com/ShuoYuyan**
- 无 AGPL 义务的专有使用
- 赔偿责任担保
- SLA / 支持合同
- 开源版中没有的定制功能

---

## Why AGPL-3.0? / 为什么选 AGPL-3.0？

We chose AGPL-3.0 to:

1. **Maximize contributor benefit** — contributors get true open-source
   rights, not source-available restrictions. They can use their work in
   any AGPL-compatible project.
2. **Prevent cloud free-riding** — AWS/GCP/Azure can't take WeDB and
   offer it as a managed service without open-sourcing their changes.
3. **Allow commercialization** — the author retains the right to sell
   commercial licenses for organizations that need proprietary use.
4. **Build a community** — AGPL is OSI-approved, widely understood, and
   used by major projects (MongoDB, Nextcloud, Red Hat, etc.).

我们选择 AGPL-3.0 是为了：

1. **最大化贡献者利益** — 贡献者获得真正的开源权利，而非源码可见但受限。
2. **防止云服务搭便车** — AWS/GCP/Azure 不能拿走 WeDB 作为托管服务而不开源。
3. **允许商业化** — 作者保留向需要专有使用的组织销售商业许可的权利。
4. **建设社区** — AGPL 是 OSI 批准的标准，被 MongoDB、Nextcloud、Red Hat 等主要项目采用。

---

## 🙏 Acknowledgments / 致谢

- The Go community for excellent tooling / Go 社区提供的优秀工具链
- The ODBC standard committee for the clear API spec / ODBC 标准委员会
- All contributors and testers / 所有贡献者和测试者
