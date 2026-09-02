$bytes = [System.IO.File]::ReadAllBytes("C:\Users\HP\Documents\WeDB\drivers\odbc\api_install.go")
$out = ""
for ($i = 0; $i -lt 100; $i++) { $out += ("{0:X2} " -f $bytes[$i]) }
$out
