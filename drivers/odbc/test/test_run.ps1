# test_run.ps1
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8
$exe = "C:\Users\HP\Documents\WeDB\drivers\odbc\test\odbc_e2e.exe"
$out = & cmd /c "$exe odbc_e2e99.db" 2>&1
$out | Out-File -FilePath C:\temp\out.log -Encoding utf8
Get-Content C:\temp\out.log
