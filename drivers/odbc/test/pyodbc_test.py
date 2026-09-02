import os, sys
# point at our private odbc config
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import pyodbc
import traceback

# We need the ODBCSYSINI env var BEFORE pyodbc connects. The setup
# script writes odbcinst.ini/odbc.ini to %TEMP%\wedb_odbc and exports
# the env var; if you run this script directly, set the env var
# beforehand or run via e2e_pyodbc.ps1.
ini_dir = os.environ.get("ODBCSYSINI", "")
print(f"ODBCSYSINI={ini_dir!r}", file=sys.stderr)

path = sys.argv[1] if len(sys.argv) > 1 else "pyodbc_test.db"
cstr = f"DRIVER={{WeDB ODBC Driver}};DBQ={path};"
print(f"Connecting: {cstr!r}")
try:
    conn = pyodbc.connect(cstr, autocommit=True)
    print("Connected")
    cur = conn.cursor()
    cur.execute("DROP TABLE IF EXISTS t")
    cur.execute("CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)")
    cur.execute("INSERT INTO t (id, name, age) VALUES (1, 'alice', 30)")
    cur.execute("INSERT INTO t (id, name, age) VALUES (2, 'bob', 25)")
    cur.commit()
    cur.execute("SELECT id, name, age FROM t ORDER BY id")
    for row in cur.fetchall():
        print("Row:", row)
    cur.execute("SELECT COUNT(*) FROM t")
    print("Count:", cur.fetchone())
    cur.execute("SELECT name FROM t WHERE age > 18")
    for row in cur.fetchall():
        print("Filter:", row)
    # metadata
    for tbl in cur.tables():
        print("Table:", tbl.table_name, tbl.table_type)
    for col in cur.columns(table='t'):
        print("Column:", col.column_name, col.type_name, col.data_type)
    print("INFO driver_name:", conn.getinfo(pyodbc.SQL_DRIVER_NAME))
    print("INFO dbms_name:", conn.getinfo(pyodbc.SQL_DBMS_NAME))
    print("INFO odbc_ver:", conn.getinfo(pyodbc.SQL_ODBC_VER))
    print("INFO txn_iso:", conn.getinfo(pyodbc.SQL_TXN_ISOLATION))
    conn.close()
    print("OK")
except Exception as e:
    print("Error:", e)
    traceback.print_exc()
