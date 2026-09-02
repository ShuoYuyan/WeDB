$lines = Get-Content "C:\Users\HP\Documents\WeDB\drivers\odbc\test\e2e.ps1"
"Line 152: [$($lines[151])]"
"Char codes: $([System.Text.Encoding]::ASCII.GetBytes($lines[151]) -join ',')"
"Length: $($lines[151].Length)"
