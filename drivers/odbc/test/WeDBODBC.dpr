program WeDBODBC;

{$APPTYPE CONSOLE}

uses
  Windows, SysUtils;

const
  SQL_HANDLE_ENV      = 1;
  SQL_HANDLE_DBC      = 2;
  SQL_HANDLE_STMT     = 3;
  SQL_NTS             = -3;
  SQL_SUCCESS         = 0;
  SQL_SUCCESS_WITH_INFO = 1;
  SQL_NO_DATA         = 100;
  SQL_ERROR           = -1;
  SQL_ATTR_ODBC_VERSION = 200;
  SQL_OV_ODBC3        = 3;
  SQL_CLOSE           = 0;
  SQL_DROP            = 1;
  SQL_C_CHAR          = 1;
  SQL_C_LONG          = 4;
  SQL_ALL_TYPES       = 0;
  SQL_DBMS_NAME       = 17;
  SQL_DRIVER_NAME     = 6;
  SQL_ODBC_VER        = 113;
  SQL_TXN_ISOLATION   = 26;
  SQL_API_SQLEXECDIRECT = 11;
  SQL_DRIVER_NOPROMPT = 0;

type
  SQLRETURN    = Integer;
  SQLINTEGER   = Integer;
  SQLSMALLINT  = SmallInt;
  SQLPOINTER   = Pointer;
  SQLLEN       = Int64;
  SQLULEN      = UInt64;
  SQLHANDLE    = NativeInt;

  TSQLAllocHandle = function(HandleType: SQLSMALLINT; InputHandle: SQLHANDLE; OutputHandle: Pointer): SQLRETURN; stdcall;
  TSQLSetEnvAttr = function(EnvironmentHandle: SQLHANDLE; Attribute: SQLINTEGER; ValuePtr: SQLPOINTER; StringLength: SQLINTEGER): SQLRETURN; stdcall;
  TSQLAllocConnect = function(EnvironmentHandle: SQLHANDLE; ConnectionHandle: Pointer): SQLRETURN; stdcall;
  TSQLAllocStmt = function(ConnectionHandle: SQLHANDLE; StatementHandle: Pointer): SQLRETURN; stdcall;
  TSQLDriverConnect = function(ConnectionHandle: SQLHANDLE; WindowHandle: HWND; InConnectionString: PAnsiChar; StringLength1: SQLSMALLINT;
    OutConnectionString: PAnsiChar; BufferLength: SQLSMALLINT; StringLength2Ptr: Pointer; DriverCompletion: Word): SQLRETURN; stdcall;
  TSQLExecDirect = function(StatementHandle: SQLHANDLE; StatementText: PAnsiChar; TextLength: SQLINTEGER): SQLRETURN; stdcall;
  TSQLPrepare = function(StatementHandle: SQLHANDLE; StatementText: PAnsiChar; TextLength: SQLINTEGER): SQLRETURN; stdcall;
  TSQLExecute = function(StatementHandle: SQLHANDLE): SQLRETURN; stdcall;
  TSQLFetch = function(StatementHandle: SQLHANDLE): SQLRETURN; stdcall;
  TSQLGetData = function(StatementHandle: SQLHANDLE; ColumnNumber: Word; TargetType: SQLSMALLINT; TargetValuePtr: Pointer;
    BufferLength: SQLLEN; StrLen_or_IndPtr: Pointer): SQLRETURN; stdcall;
  TSQLRowCount = function(StatementHandle: SQLHANDLE; RowCount: Pointer): SQLRETURN; stdcall;
  TSQLNumResultCols = function(StatementHandle: SQLHANDLE; ColumnCount: Pointer): SQLRETURN; stdcall;
  TSQLDescribeCol = function(StatementHandle: SQLHANDLE; ColumnNumber: Word; ColumnName: PAnsiChar; BufferLength: SQLSMALLINT;
    NameLength: Pointer; DataType: Pointer; ColumnSize: Pointer; DecimalDigits: Pointer; Nullable: Pointer): SQLRETURN; stdcall;
  TSQLFreeStmt = function(StatementHandle: SQLHANDLE; Option: Word): SQLRETURN; stdcall;
  TSQLTables = function(StatementHandle: SQLHANDLE; CatalogName: PAnsiChar; NameLength1: SQLSMALLINT; SchemaName: PAnsiChar; NameLength2: SQLSMALLINT;
    TableName: PAnsiChar; NameLength3: SQLSMALLINT; TableType: PAnsiChar; NameLength4: SQLSMALLINT): SQLRETURN; stdcall;
  TSQLColumns = function(StatementHandle: SQLHANDLE; CatalogName: PAnsiChar; NameLength1: SQLSMALLINT; SchemaName: PAnsiChar; NameLength2: SQLSMALLINT;
    TableName: PAnsiChar; NameLength3: SQLSMALLINT; ColumnName: PAnsiChar; NameLength4: SQLSMALLINT): SQLRETURN; stdcall;
  TSQLGetTypeInfo = function(StatementHandle: SQLHANDLE; DataType: SQLSMALLINT): SQLRETURN; stdcall;
  TSQLGetInfo = function(ConnectionHandle: SQLHANDLE; InfoType: Word; InfoValuePtr: Pointer;
    BufferLength: SQLSMALLINT; StringLengthPtr: Pointer): SQLRETURN; stdcall;
  TSQLGetFunctions = function(ConnectionHandle: SQLHANDLE; FunctionId: Word; Supported: Pointer): SQLRETURN; stdcall;
  TSQLGetDiagRec = function(HandleType: SQLSMALLINT; Handle: SQLHANDLE; RecNumber: SQLSMALLINT; SQLState: PAnsiChar;
    NativeErrorPtr: Pointer; MessageText: PAnsiChar; BufferLength: SQLSMALLINT; TextLengthPtr: Pointer): SQLRETURN; stdcall;
  TSQLDisconnect = function(ConnectionHandle: SQLHANDLE): SQLRETURN; stdcall;
  TSQLFreeHandle = function(HandleType: SQLSMALLINT; Handle: SQLHANDLE): SQLRETURN; stdcall;

var
  hDLL: THandle;
  pAllocHandle: TSQLAllocHandle;
  pSetEnvAttr: TSQLSetEnvAttr;
  pAllocConnect: TSQLAllocConnect;
  pAllocStmt: TSQLAllocStmt;
  pDriverConnect: TSQLDriverConnect;
  pExecDirect: TSQLExecDirect;
  pPrepare: TSQLPrepare;
  pExecute: TSQLExecute;
  pFetch: TSQLFetch;
  pGetData: TSQLGetData;
  pRowCount: TSQLRowCount;
  pNumResultCols: TSQLNumResultCols;
  pDescribeCol: TSQLDescribeCol;
  pFreeStmt: TSQLFreeStmt;
  pTables: TSQLTables;
  pColumns: TSQLColumns;
  pGetTypeInfo: TSQLGetTypeInfo;
  pGetInfo: TSQLGetInfo;
  pGetFunctions: TSQLGetFunctions;
  pGetDiagRec: TSQLGetDiagRec;
  pDisconnect: TSQLDisconnect;
  pFreeHandle: TSQLFreeHandle;
  env, dbc, stmt: SQLHANDLE;
  DbPath: string;
  Conn: string;
  OutBuf: array[0..1023] of AnsiChar;
  OutLen: SQLSMALLINT;
  ExecRC: SQLRETURN;
  Rows: SQLLEN;
  NCols: SQLSMALLINT;
  i: Integer;
  Fetched, tblCount, colCount, n: Integer;
  ColName: array[0..63] of AnsiChar;
  NameLen, DataType, Nullable, Dec: SQLSMALLINT;
  ColSize: SQLULEN;
  id, age: SQLLEN;
  name: array[0..63] of AnsiChar;
  buf: array[0..4] of array[0..63] of AnsiChar;
  ind: SQLLEN;
  cat, schem, tblName, typ: array[0..31] of AnsiChar;
  col, typeName: array[0..63] of AnsiChar;
  colDataType: SQLINTEGER;
  infoBuf: array[0..63] of AnsiChar;
  infoLen: SQLSMALLINT;
  iso: SQLINTEGER;
  supp: Word;
  dummyVal: SQLINTEGER;
  dummyPtr: Pointer;

function BufToStr(const Buf; BufLen: Integer): string;
var
  P: PAnsiChar;
  i: Integer;
  found: Boolean;
begin
  P := PAnsiChar(@Buf);
  for i := 0 to BufLen - 1 do
    if P[i] = #0 then begin Result := Copy(P, 1, i); Exit; end;
  Result := Copy(P, 1, BufLen);
end;

function GetProc(const Dll: THandle; const Name: string): Pointer;
var
  AnsiName: AnsiString;
begin
  AnsiName := AnsiString(Name);
  Result := GetProcAddress(Dll, PAnsiChar(AnsiName));
  if not Assigned(Result) then
  begin
    Writeln(Format('missing: %s (err=%d)', [Name, GetLastError]));
    Halt(1);
  end;
end;

procedure Check(RC: SQLRETURN; const What: string);
begin
  if (RC = SQL_SUCCESS) or (RC = SQL_SUCCESS_WITH_INFO) then
  begin
    Writeln('  OK ', What);
    Exit;
  end;
  if RC = SQL_NO_DATA then
  begin
    Writeln('  OK (no data) ', What);
    Exit;
  end;
  Writeln('  FAIL ', What, ' rc=', RC);
  Halt(1);
end;

begin
  try
    if ParamCount < 1 then
    begin
      Writeln('Usage: WeDBODBC <path-to-db>');
      Halt(2);
    end;
    DbPath := ParamStr(1);

    hDLL := LoadLibraryA('wedb_odbc.dll');
    if hDLL = 0 then
    begin
      Writeln('LoadLibrary failed: ', GetLastError);
      Halt(1);
    end;
    Writeln('DLL: ', IntToHex(hDLL, 16));

    pAllocHandle := TSQLAllocHandle(GetProc(hDLL, 'SQLAllocHandle'));
    pSetEnvAttr := TSQLSetEnvAttr(GetProc(hDLL, 'SQLSetEnvAttr'));
    pAllocConnect := TSQLAllocConnect(GetProc(hDLL, 'SQLAllocConnect'));
    pAllocStmt := TSQLAllocStmt(GetProc(hDLL, 'SQLAllocStmt'));
    pDriverConnect := TSQLDriverConnect(GetProc(hDLL, 'SQLDriverConnect'));
    pExecDirect := TSQLExecDirect(GetProc(hDLL, 'SQLExecDirect'));
    pPrepare := TSQLPrepare(GetProc(hDLL, 'SQLPrepare'));
    pExecute := TSQLExecute(GetProc(hDLL, 'SQLExecute'));
    pFetch := TSQLFetch(GetProc(hDLL, 'SQLFetch'));
    pGetData := TSQLGetData(GetProc(hDLL, 'SQLGetData'));
    pRowCount := TSQLRowCount(GetProc(hDLL, 'SQLRowCount'));
    pNumResultCols := TSQLNumResultCols(GetProc(hDLL, 'SQLNumResultCols'));
    pDescribeCol := TSQLDescribeCol(GetProc(hDLL, 'SQLDescribeCol'));
    pFreeStmt := TSQLFreeStmt(GetProc(hDLL, 'SQLFreeStmt'));
    pTables := TSQLTables(GetProc(hDLL, 'SQLTables'));
    pColumns := TSQLColumns(GetProc(hDLL, 'SQLColumns'));
    pGetTypeInfo := TSQLGetTypeInfo(GetProc(hDLL, 'SQLGetTypeInfo'));
    pGetInfo := TSQLGetInfo(GetProc(hDLL, 'SQLGetInfo'));
    pGetFunctions := TSQLGetFunctions(GetProc(hDLL, 'SQLGetFunctions'));
    pGetDiagRec := TSQLGetDiagRec(GetProc(hDLL, 'SQLGetDiagRec'));
    pDisconnect := TSQLDisconnect(GetProc(hDLL, 'SQLDisconnect'));
    pFreeHandle := TSQLFreeHandle(GetProc(hDLL, 'SQLFreeHandle'));

    env := 0; dbc := 0; stmt := 0;
    Check(pAllocHandle(SQL_HANDLE_ENV, 0, @env), 'AllocEnv');
    dummyVal := SQL_OV_ODBC3;
    Check(pSetEnvAttr(env, SQL_ATTR_ODBC_VERSION, @dummyVal, 0), 'SetEnvAttr');
    Check(pAllocConnect(env, @dbc), 'AllocConnect');

    Conn := 'DRIVER={WeDB ODBC Driver};DBQ=' + DbPath + ';';
    StrPCopy(@OutBuf[0], '');
    OutLen := 0;
    Check(pDriverConnect(dbc, 0, PAnsiChar(AnsiString(Conn)), SQL_NTS, @OutBuf[0], Length(OutBuf), @OutLen, SQL_DRIVER_NOPROMPT), 'DriverConnect');
    Writeln('  out: ', BufToStr(OutBuf, Length(OutBuf)));

    Check(pAllocStmt(dbc, @stmt), 'AllocStmt');

    Check(pExecDirect(stmt, 'DROP TABLE IF EXISTS t', SQL_NTS), 'DROP TABLE');
    Check(pExecDirect(stmt, 'CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)', SQL_NTS), 'CREATE TABLE');
    Check(pExecDirect(stmt, 'INSERT INTO t (id, name, age) VALUES (1, ''alice'', 30)', SQL_NTS), 'INSERT 1');
    Check(pExecDirect(stmt, 'INSERT INTO t (id, name, age) VALUES (2, ''bob'', 25)', SQL_NTS), 'INSERT 2');
    Check(pExecDirect(stmt, 'INSERT INTO t (id, name, age) VALUES (3, ''carol'', 40)', SQL_NTS), 'INSERT 3');

    Rows := 0;
    pRowCount(stmt, @Rows);
    Writeln('  RowCount after INSERT: ', Rows);

    Check(pPrepare(stmt, 'SELECT id, name, age FROM t WHERE id = ?', SQL_NTS), 'PREPARE');
    ExecRC := pExecute(stmt);
    if ExecRC = SQL_ERROR then
      Writeln('  (prepared execute returned error -- expected for unbound ?)')
    else
      Writeln('  prepared execute rc=', ExecRC);
    pFreeStmt(stmt, SQL_CLOSE);

    Check(pExecDirect(stmt, 'SELECT id, name, age FROM t ORDER BY id', SQL_NTS), 'SELECT');
    NCols := 0;
    pNumResultCols(stmt, @NCols);
    Writeln('  columns: ', NCols);
    for i := 1 to NCols do
    begin
      FillChar(ColName, SizeOf(ColName), 0);
      NameLen := 0; DataType := 0; ColSize := 0; Dec := 0; Nullable := 0;
      pDescribeCol(stmt, i, @ColName[0], SQLSMALLINT(Length(ColName)), @NameLen, @DataType, @ColSize, @Dec, @Nullable);
      Writeln(Format('    [%d] %s type=%d nameLen=%d', [i, BufToStr(ColName, Length(ColName)), DataType, NameLen]));
    end;

    Fetched := 0;
    while pFetch(stmt) = SQL_SUCCESS do
    begin
      FillChar(name, SizeOf(name), 0);
      ind := 0;
      pGetData(stmt, 1, SQL_C_LONG, @id, 0, @ind);
      pGetData(stmt, 2, SQL_C_CHAR, @name[0], SQLSMALLINT(Length(name)), @ind);
      pGetData(stmt, 3, SQL_C_LONG, @age, 0, @ind);
      Writeln(Format('  row: id=%d name=%s age=%d', [id, BufToStr(name, Length(name)), age]));
      Inc(Fetched);
    end;
    Writeln('  total rows fetched: ', Fetched);

    pFreeStmt(stmt, SQL_CLOSE);

    Check(pExecDirect(stmt, 'SELECT COUNT(*), SUM(age), AVG(age), MIN(age), MAX(age) FROM t', SQL_NTS), 'SELECT aggregate');
    if pFetch(stmt) = SQL_SUCCESS then
    begin
      for i := 0 to 4 do
      begin
        FillChar(buf[i], SizeOf(buf[i]), 0);
        ind := 0;
        pGetData(stmt, i + 1, SQL_C_CHAR, @buf[i][0], SQLSMALLINT(Length(buf[i])), @ind);
      end;
      Writeln(Format('  COUNT=%s SUM=%s AVG=%s MIN=%s MAX=%s',
        [BufToStr(buf[0], Length(buf[0])), BufToStr(buf[1], Length(buf[1])),
         BufToStr(buf[2], Length(buf[2])), BufToStr(buf[3], Length(buf[3])),
         BufToStr(buf[4], Length(buf[4]))]));
    end;

    pFreeStmt(stmt, SQL_CLOSE);

    Check(pExecDirect(stmt, 'SELECT name FROM t WHERE age > 18 AND name = ''alice''', SQL_NTS), 'SELECT WHERE');
    while pFetch(stmt) = SQL_SUCCESS do
    begin
      FillChar(name, SizeOf(name), 0);
      ind := 0;
      pGetData(stmt, 1, SQL_C_CHAR, @name[0], SQLSMALLINT(Length(name)), @ind);
      Writeln('  WHERE match: ', BufToStr(name, Length(name)));
    end;

    pFreeStmt(stmt, SQL_CLOSE);

    Check(pTables(stmt, nil, 0, nil, 0, nil, SQL_NTS, nil, 0), 'SQLTables');
    tblCount := 0;
    while pFetch(stmt) = SQL_SUCCESS do
    begin
      FillChar(cat, SizeOf(cat), 0);
      FillChar(schem, SizeOf(schem), 0);
      FillChar(tblName, SizeOf(tblName), 0);
      FillChar(typ, SizeOf(typ), 0);
      ind := 0;
      pGetData(stmt, 1, SQL_C_CHAR, @cat[0], SQLSMALLINT(Length(cat)), @ind);
      pGetData(stmt, 2, SQL_C_CHAR, @schem[0], SQLSMALLINT(Length(schem)), @ind);
      pGetData(stmt, 3, SQL_C_CHAR, @tblName[0], SQLSMALLINT(Length(tblName)), @ind);
      pGetData(stmt, 4, SQL_C_CHAR, @typ[0], SQLSMALLINT(Length(typ)), @ind);
      Writeln(Format('  Table: %s.%s.%s type=%s',
        [BufToStr(cat, Length(cat)), BufToStr(schem, Length(schem)),
         BufToStr(tblName, Length(tblName)), BufToStr(typ, Length(typ))]));
      Inc(tblCount);
    end;
    Writeln('  tables: ', tblCount);

    pFreeStmt(stmt, SQL_CLOSE);

    Check(pColumns(stmt, nil, 0, nil, 0, 't', SQL_NTS, nil, 0), 'SQLColumns(t)');
    colCount := 0;
    while pFetch(stmt) = SQL_SUCCESS do
    begin
      FillChar(col, SizeOf(col), 0);
      FillChar(typeName, SizeOf(typeName), 0);
      colDataType := 0;
      ind := 0;
      pGetData(stmt, 4, SQL_C_CHAR, @col[0], SQLSMALLINT(Length(col)), @ind);
      pGetData(stmt, 5, SQL_C_LONG, @colDataType, 0, @ind);
      pGetData(stmt, 6, SQL_C_CHAR, @typeName[0], SQLSMALLINT(Length(typeName)), @ind);
      Writeln(Format('  Column: %s type=%d (%s)',
        [BufToStr(col, Length(col)), colDataType, BufToStr(typeName, Length(typeName))]));
      Inc(colCount);
    end;
    Writeln('  columns for t: ', colCount);

    pFreeStmt(stmt, SQL_CLOSE);

    FillChar(infoBuf, SizeOf(infoBuf), 0);
    pGetInfo(dbc, SQL_DBMS_NAME, @infoBuf[0], SQLSMALLINT(Length(infoBuf)), @infoLen);
    Writeln('  DBMS_NAME=', BufToStr(infoBuf, Length(infoBuf)));
    FillChar(infoBuf, SizeOf(infoBuf), 0);
    pGetInfo(dbc, SQL_DRIVER_NAME, @infoBuf[0], SQLSMALLINT(Length(infoBuf)), @infoLen);
    Writeln('  DRIVER_NAME=', BufToStr(infoBuf, Length(infoBuf)));
    FillChar(infoBuf, SizeOf(infoBuf), 0);
    pGetInfo(dbc, SQL_ODBC_VER, @infoBuf[0], SQLSMALLINT(Length(infoBuf)), @infoLen);
    Writeln('  ODBC_VER=', BufToStr(infoBuf, Length(infoBuf)));
    iso := 0;
    pGetInfo(dbc, SQL_TXN_ISOLATION, @iso, SQLSMALLINT(SizeOf(iso)), @infoLen);
    Writeln('  TXN_ISOLATION=', iso);

    supp := 0;
    pGetFunctions(dbc, SQL_API_SQLEXECDIRECT, @supp);
    Writeln('  SQLExecDirect supported: ', supp);

    pFreeStmt(stmt, SQL_CLOSE);

    Check(pGetTypeInfo(stmt, SQL_ALL_TYPES), 'SQLGetTypeInfo');
    n := 0;
    while (pFetch(stmt) = SQL_SUCCESS) and (n < 5) do
    begin
      FillChar(name, SizeOf(name), 0);
      colDataType := 0;
      ind := 0;
      pGetData(stmt, 1, SQL_C_CHAR, @name[0], SQLSMALLINT(Length(name)), @ind);
      pGetData(stmt, 2, SQL_C_LONG, @colDataType, 0, @ind);
      Writeln(Format('  Type: %s (%d)', [BufToStr(name, Length(name)), colDataType]));
      Inc(n);
    end;
    Writeln('  types shown: ', n);

    pFreeStmt(stmt, SQL_DROP);
    pDisconnect(dbc);
    pFreeHandle(SQL_HANDLE_DBC, dbc);
    pFreeHandle(SQL_HANDLE_ENV, env);
    FreeLibrary(hDLL);
    DeleteFile(PAnsiChar(AnsiString(DbPath)));
    Writeln('OK');
  except
    on E: Exception do
    begin
      Writeln('ERROR: ', E.Message);
      Halt(1);
    end;
  end;
end.
