$body = '{"username":"admin","password":"Admin@123"}'
$login = Invoke-RestMethod -Uri "http://localhost:2091/api/v1/auth/login" -Method Post -Body $body -ContentType "application/json"
$token = $login.data.token

$raw = Invoke-WebRequest -Uri "http://localhost:2091/api/v1/reports/yoy-comparison/donors?month=8&year=2026" -Headers @{ Authorization = "Bearer $token" }
Write-Output $raw.Content.Substring(0, 300)
