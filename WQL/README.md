# WQL - WeDB Native Query Language

## 姒傝堪

WQL 鏄?WeDB 鐨?*鍘熺敓鏌ヨ璇█**锛屽畬鍏ㄧ嫭绔嬪疄鐜扮殑鏌ヨ璁″垝鍣ㄣ€?WQL 涓嶇敓鎴愪换浣?SQL 瀛楃涓?鈥斺€?瀹冪洿鎺ヨ皟鐢?WeDB 鐨?Go API锛坄storage.WeDBDatabase`锛夋潵鎵ц鏌ヨ銆?
## 璁捐鍘熷垯

1. **闆?SQL 瀛楃涓?*锛歐QL 寮曟搸浠庝笉鐢熸垚鎴栨墽琛?SQL 璇彞
2. **鐩存帴璋冪敤瀛樺偍 API**锛氶€氳繃 `Adapter` 鎺ュ彛鐩存帴璋冪敤 WeDB 鐨?Go 鏂规硶
3. **瀹屽叏绫诲瀷鍖?*锛氫娇鐢?Go 绫诲瀷绯荤粺琛ㄨ揪鎵€鏈夎〃杈惧紡鍜屾搷浣?4. **鍙墿灞?*锛氭湭鏉ュ彲瀹炵幇 `PostgreSQLAdapter`銆乣MySQLAdapter` 绛?
## 鏋舵瀯

```
鈹屸攢鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹?鈹? CLI / REPL      鈹? cmd/wql/
鈹? (瀛楃涓叉帴鍙?     鈹?鈹斺攢鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹攢鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹?         鈹?瀛楃涓茶В鏋?         鈻?鈹屸攢鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹?鈹? wqlv3.QueryBuilder 鈹? pkg/wqlv3/wqlv3.go
鈹? (閾惧紡 Go API)     鈹? db.Table("users").Where("age > 18").All()
鈹斺攢鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹攢鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹?         鈹?Expression AST
         鈻?鈹屸攢鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹?鈹? Expression      鈹? pkg/wqlv3/expression.go
鈹? (琛ㄨ揪寮忔眰鍊?     鈹? ParseWhere() + EvalBoolExpr()
鈹斺攢鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹攢鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹?         鈹?Adapter interface
         鈻?鈹屸攢鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹?鈹? WeDBAdapter     鈹? pkg/wqlv3/wedb_adapter.go
鈹? (WeDB 閫傞厤鍣?    鈹? 鐩存帴璋冪敤 WeDB Go API
鈹斺攢鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹攢鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹?         鈹?         鈻?鈹屸攢鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹?鈹? WeDB Storage    鈹? internal/storage/
鈹? (B-Tree + MVCC)  鈹?鈹斺攢鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹€鈹?```

## WQL 璇硶

WQL 鎻愪緵**涓ょ**浣跨敤鏂瑰紡锛?
### 鏂瑰紡 1: Go 閾惧紡 API锛堟帹鑽愮敤浜?Go 绋嬪簭锛?
```go
import "github.com/wedb/wedb/WQL/pkg/wqlv3"

db := storage.NewWeDBDatabase("test.db", 4096)
adapter := wqlv3.NewWeDBAdapter(db)
wdb := wqlv3.NewDatabase(adapter)

// 鍏ㄨ〃鏌ヨ
rows, _ := wdb.Table("users").All()

// 鏉′欢鏌ヨ
rows, _ := wdb.Table("users").Where("age > 18").All()

// 閫夋嫨鍒?rows, _ := wdb.Table("users").Select("id", "name").Where("age > 18").All()

// 鎺掑簭 + 鍒嗛〉
rows, _ := wdb.Table("users").OrderBy("age", "DESC").Skip(10).Take(20).All()

// 鑱氬悎
count, _ := wdb.Table("users").Where("age > 18").Count()
total, _ := wdb.Table("orders").Sum("amount")
avg, _ := wdb.Table("orders").Avg("amount")

// 绗竴琛?row, _ := wdb.Table("users").Where("email = 'alice@x.com'").First()
```

### 鏂瑰紡 2: WQL 鏃犲弻寮曞彿瀛楃涓诧紙鎺ㄨ崘 鈥?鐪熸鐨?WQL 璇硶锛?
WQL 鐨勮璁″師鍒欐槸**鏃犲弻寮曞彿**锛氭爣璇嗙锛堣〃鍚嶃€佸垪鍚嶏級涓嶉渶瑕佸紩鍙枫€?
```go
result, err := wqlv3.EvaluateQueryNoQuotes(wdb, `db.Table(users).Where(age > 18).All()`)
result, err := wqlv3.EvaluateQueryNoQuotes(wdb, `db.Table(orders).Select(city, Count()).GroupBy(city).All()`)
result, err := wqlv3.EvaluateQueryNoQuotes(wdb, `db.Table(users).Where(name = "alice").First()`)
```

**璇硶瑕佺偣**:
- 琛ㄥ悕銆佸垪鍚?*涓嶅姞寮曞彿**锛歚db.Table(users)`, `Select(name, age)`
- 瀛楃涓插€奸渶瑕佸紩鍙凤細`name = "alice"`
- 鏁板瓧涓嶅姞寮曞彿锛歚age > 18`
- 鎿嶄綔閾撅細`db.Table(t).Select(...).Where(...).OrderBy(..., DESC).Take(N).All()`

### 鏂瑰紡 3: 鏃у瓧绗︿覆鎺ュ彛锛堝悜鍚庡吋瀹癸紝甯﹀弻寮曞彿锛?
```go
result, err := wqlv3.EvaluateQuery(wdb, `T("users").Where("age > 18").All()`)
result, err := wqlv3.EvaluateQuery(wdb, `T("orders").Count()`)
```

## CLI 鐢ㄦ硶

### 鍚姩 REPL
```cmd
> wql-cli test.db

 鈺?鈺︹晹鈺愨晽鈺? 鈺斺晲鈺? 鈺戔晳鈺戔晳鈺?鈺? 鈺犫晲鈺? 鈺氣暕鈺濃暁鈺愨暆鈺┾晲鈺濃暕 鈺?WeDB Native Query Language

 v0.1.0  鈥? backed by WeDB pure-Go storage engine
 type 'help' for commands, 'quit' to exit

  database: test.db
  backend:  wqlv3 + WeDB native Go storage

wql> help
Available commands:
  tables              - List all tables
  schema <table>      - Show table schema
  help                - Show this help
  quit / exit          - Exit the REPL

WQL Syntax (fluent API, no SQL strings):
  T(<table>)                 - Reference a table
  .Select(col1, col2, ...)   - Select columns
  .Where(condition)          - Filter rows
  .OrderBy(col, "ASC|DESC")  - Sort results
  .Take(n)                   - Limit rows
  .Skip(n)                   - Offset rows
  .All()                     - Execute and return []map[string]any
  .First()                   - Return first row
  .Count()                   - Count rows
  .Sum(col) / .Avg(col)      - Aggregations

wql> tables
  Found 1 table(s):
    - users

wql> db.Table(users).All()
  id  name  age
  --  ----  ---
  1   alice 30
  2   bob   25
  3   carol 40

  3 row(s) in 0.123ms

wql> db.Table(users).Select(name, age).Where(age > 18).OrderBy(age, DESC).Take(2).All()
  name  age
  ----  ---
  carol 40
  alice 30

wql> db.Table(users).Where(name = "alice").First()
  id  name  age
  --  ----  ---
  1   alice 30

wql> quit
Bye!
```

> **璁捐鍘熷垯**: WQL 浣跨敤**鏃犲弻寮曞彿**璇硶 鈥?鏍囪瘑绗︼紙琛ㄥ悕銆佸垪鍚嶏級涓嶉渶瑕佸紩鍙凤紝鍙湁瀛楃涓插€兼墠闇€瑕併€?> 渚? `db.Table(users).Select(name, age).Where(name = "alice").All()`
> 鑰?*涓嶆槸**: `db.Table("users").Select("name", "age").Where("name = \"alice\"").All()`

### 鍗曟鏌ヨ妯″紡
```cmd
> wql-cli test.db 'T("users").Count()'
> wql-cli test.db 'T("users").All()'
> wql-cli test.db 'T("users").Where("age > 18").All()'
```

## DML / DDL 鏃犲弻寮曞彿璇硶

WQL v3 鏀寔瀹屾暣鐨?DML/DDL 閾惧紡璇硶锛屽叏閮ㄦ棤闇€鍙屽紩鍙凤細

### CREATE TABLE / DROP TABLE

```go
// 鍒涘缓琛?_, err := wqlv3.EvaluateQueryNoQuotes(wdb, `
    db.Table(products).Create(id INTEGER PRIMARY KEY, name TEXT, price REAL).Execute()
`)

// 鍒犻櫎琛?_, err := wqlv3.EvaluateQueryNoQuotes(wdb, `db.Table(temp).Drop().Execute()`)
```

鏀寔绫诲瀷锛歚INTEGER`, `TEXT`, `REAL`, `BLOB`锛坄INT`/`VARCHAR`/`FLOAT`/`DOUBLE` 涔熷彲璇嗗埆锛?鏀寔绾︽潫锛歚PRIMARY KEY`, `NOT NULL`, `NULL`

### INSERT

```go
// 瀵硅薄瀛楅潰閲忓舰寮?_, err := wqlv3.EvaluateQueryNoQuotes(wdb, `
    db.Table(users).Insert({id: 1, name: "alice", age: 30}).Execute()
`)

// 鍒楀€煎褰㈠紡
_, err := wqlv3.EvaluateQueryNoQuotes(wdb, `
    db.Table(users).Insert(id, 2, name, "bob", age, 25).Execute()
`)

// 澶氳
_, err := wqlv3.EvaluateQueryNoQuotes(wdb, `
    db.Table(users).Insert(
        {id: 1, name: "alice", age: 30},
        {id: 2, name: "bob", age: 25}
    ).Execute()
`)
```

### UPDATE (Set + Where)

```go
_, err := wqlv3.EvaluateQueryNoQuotes(wdb, `
    db.Table(users).Set(age, 31).Where(id = 1).Execute()
`)
```

### DELETE (Where + Delete)

```go
_, err := wqlv3.EvaluateQueryNoQuotes(wdb, `
    db.Table(users).Where(age < 18).Delete().Execute()
`)
```

### 瀹屾暣鐢熷懡鍛ㄦ湡绀轰緥

```go
// 1. 寤鸿〃
wqlv3.EvaluateQueryNoQuotes(wdb, `db.Table(orders).Create(
    id INTEGER PRIMARY KEY, product TEXT, qty INTEGER
).Execute()`)

// 2. 鎻掓暟鎹?wqlv3.EvaluateQueryNoQuotes(wdb, `db.Table(orders).Insert(
    {id: 1, product: "apple", qty: 10},
    {id: 2, product: "banana", qty: 20}
).Execute()`)

// 3. 鏌?res, _ := wqlv3.EvaluateQueryNoQuotes(wdb, `db.Table(orders).All()`)

// 4. 鏀?wqlv3.EvaluateQueryNoQuotes(wdb, `db.Table(orders).Set(qty, 100).Where(product = "apple").Execute()`)

// 5. 鍒?wqlv3.EvaluateQueryNoQuotes(wdb, `db.Table(orders).Where(qty < 50).Delete().Execute()`)

// 6. 鍒犺〃
wqlv3.EvaluateQueryNoQuotes(wdb, `db.Table(orders).Drop().Execute()`)
```

## WHERE 瀛愬彞璇硶

鏀寔鐨勮繍绠楃鍜岃〃杈惧紡锛?
| 绫诲瀷 | 璇硶 | 绀轰緥 |
|---|---|---|
| 绛変簬 | `=` | `age = 18` |
| 涓嶇瓑浜?| `!=` 鎴?`<>` | `status != "active"` |
| 澶т簬 | `>` | `age > 18` |
| 澶т簬绛変簬 | `>=` | `age >= 18` |
| 灏忎簬 | `<` | `price < 100` |
| 灏忎簬绛変簬 | `<=` | `price <= 100` |
| 閫昏緫涓?| `AND` | `age > 18 AND age < 30` |
| 閫昏緫鎴?| `OR` | `status = "active" OR status = "pending"` |
| 閫昏緫闈?| `NOT` | `NOT deleted` |
| IN | `IN (a, b, c)` | `id IN (1, 2, 3)` |
| LIKE | `LIKE "pattern"` | `name LIKE "a%"` (鍏朵腑 % = 浠绘剰, _ = 鍗曞瓧绗? |
| IS NULL | `IS NULL` | `deleted_at IS NULL` |
| IS NOT NULL | `IS NOT NULL` | `email IS NOT NULL` |

## 涓嶄緷璧?SQL 鐨勮瘉鏄?
WQL 鐨勬墽琛岃矾寰?*瀹屽叏缁曡繃 SQL**锛?
| 浼犵粺 SQL 璺緞 | WQL 璺緞 |
|---|---|
| `SQLExecDirect("SELECT ...")` | `wqlv3.QueryBuilder.All()` |
| 鈫?瑙ｆ瀽 SQL 瀛楃涓?| 鈫?鐩存帴鏋勯€?Go 瀵硅薄 |
| 鈫?鐢熸垚鏌ヨ璁″垝 | 鈫?宸茬粡鏄煡璇㈣鍒?|
| 鈫?璋冪敤 B-tree | 鈫?閫氳繃 Adapter 鐩存帴璋冪敤 B-tree |

WQL 鍖?*涓嶅鍏?* `database/sql`銆傞獙璇佹柟娉曪細
```bash
$ grep -r "database/sql" pkg/wqlv3/
# 鏃犺緭鍑?```

## 娴嬭瘯

```bash
cd WQL
go test ./pkg/wqlv3/
```

娴嬭瘯瑕嗙洊锛?- WHERE 瀛愬彞瑙ｆ瀽锛?, !=, <, >, AND, OR, NOT, IN, IS NULL, LIKE锛?- QueryBuilder 鍐呭瓨杩囨护
- 鎺掑簭锛圓SC/DESC锛?- 涓嶇敓鎴?SQL 瀛楃涓茬殑璇佹槑娴嬭瘯

## 宸茬煡闄愬埗

褰撳墠 wqlv3 瀹炵幇鐨勫姛鑳斤細
- 鉁?SELECT锛堝甫鍒楄繃婊ゃ€乄HERE銆丱RDER BY銆丼KIP銆乀AKE锛?- 鉁?INSERT / UPDATE / DELETE锛圖ML 瀹屾暣鏀寔锛?- 鉁?CREATE TABLE / DROP TABLE锛圖DL 鍩虹鏀寔锛?  - 鉁?鑱氬悎锛圕ount, Sum, Avg, Min, Max 鈥?閫氳繃 Go API锛?  - 鉁?WHERE 杩囨护锛?, !=, <, >, AND, OR, NOT, IN, LIKE, IS NULL, IS NOT NULL锛?  - 鉁?WQL 鏃犲弻寮曞彿瑙ｆ瀽鍣紙lexer + parser + AST锛?  - 鉁?DML: Insert / Update (Set+Where) / Delete (Where+Delete)
  - 鉁?DDL: CreateTable / DropTable
  - 鉁?瀵硅薄瀛楅潰閲?`{col: val, ...}` 鐢ㄤ簬 Insert/Set
  - 鉂?JOIN锛坧arser 鏆備笉鏀寔锛?  - 鉂?GROUP BY / HAVING锛坧arser 鏆備笉鏀寔锛?- 鉂?宓屽瀛愭煡璇紙parser 鏆備笉鏀寔锛?- 鉂?绐楀彛鍑芥暟锛堝緟瀹炵幇锛?
## 椤圭洰鍘嗗彶

WQL 缁忓巻浜嗕笁涓富瑕佺増鏈細
- **v1**锛堝湪 `_attic_pkg_wql/`锛夛細鏈€鍒濆疄鐜帮紝鍩轰簬 SQLite 鍚庣
- **v2**锛堝湪 `_attic_pkg_wqlv2/`锛夛細閲嶆瀯灏濊瘯
- **v3**锛堝綋鍓嶅湪 `pkg/wqlv3/`锛夛細瀹屽叏閲嶅啓锛屽熀浜?WeDB 鍘熺敓 Go API锛?*涓嶄緷璧栦换浣?SQL**

v3 鏄綋鍓嶆寮忕増鏈€倂1 鍜?v2 宸插綊妗ｄ繚鐣欎緵鍙傝€冦€?
## 鏂囦欢缁撴瀯

```
WQL/
鈹溾攢鈹€ go.mod                              # 妯″潡瀹氫箟
鈹溾攢鈹€ README.md                           # 鏈枃浠?鈹溾攢鈹€ cmd/
鈹?  鈹斺攢鈹€ wql/
鈹?      鈹斺攢鈹€ main.go                     # CLI 鍏ュ彛
鈹溾攢鈹€ pkg/
鈹?  鈹斺攢鈹€ wqlv3/                          # WQL v3 姝ｅ紡鐗?鈹?      鈹溾攢鈹€ wqlv3.go                    # QueryBuilder (Fluent API)
鈹?      鈹溾攢鈹€ expression.go                # Expression AST + ParseWhere
鈹?      鈹溾攢鈹€ expression_test.go           # 鍗曞厓娴嬭瘯
鈹?      鈹溾攢鈹€ wedb_adapter.go              # WeDB Adapter 瀹炵幇
鈹?      鈹斺攢鈹€ cli_helpers.go              # CLI 杈呭姪鍑芥暟
鈹溾攢鈹€ _attic_pkg_wql/                      # 褰掓。: WQL v1 (SQLite 鍚庣)
鈹溾攢鈹€ _attic_pkg_wqlv2/                   # 褰掓。: WQL v2
鈹溾攢鈹€ _attic_examples/                    # 褰掓。: 鏃хず渚?鈹溾攢鈹€ _attic_tools/                        # 褰掓。: 鏃у伐鍏?鈹溾攢鈹€ _attic_verification/                 # 褰掓。: 鏃ч獙璇?鈹溾攢鈹€ _attic_wql-editor/                   # 褰掓。: PyQt5 IDE
鈹溾攢鈹€ _attic_cmd_*/                       # 褰掓。: 鏃ф祴璇曠▼搴?```

## 璁稿彲

WQL 涓?WeDB 涓€鏍凤紝閬靛惊 AGPL-3.0 鍗忚銆傝瑙佹牴鐩綍鐨?`LICENSE` 鏂囦欢銆?