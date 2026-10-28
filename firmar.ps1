# Compila y firma organizador_carpetas.exe con el certificado S4M-Certificate.pfx
#
# Uso (en PowerShell, dentro de la carpeta del proyecto):
#   .\firmar.ps1                 -> compila con wails y firma
#   .\firmar.ps1 -SinCompilar    -> solo firma el .exe que ya existe
#
# Si PowerShell no deja ejecutar scripts:  powershell -ExecutionPolicy Bypass -File .\firmar.ps1

param(
    [string]$Certificado = (Join-Path $PSScriptRoot 'S4M-Certificate.pfx'),
    [switch]$SinCompilar
)
$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

# 1. Compilar (WebView2 incluido, por si el equipo no lo tiene)
if (-not $SinCompilar) {
    Write-Host '== Compilando con wails ==' -ForegroundColor Cyan
    wails build -webview2 embed
    if ($LASTEXITCODE -ne 0) { throw 'wails build falló' }
}

$exe = Join-Path $PSScriptRoot 'build\bin\organizador_carpetas.exe'
if (-not (Test-Path $exe)) { throw "No se encontró $exe" }
if (-not (Test-Path $Certificado)) { throw "No se encontró el certificado $Certificado" }

# 2. Cargar el certificado (pide la clave del .pfx)
Write-Host '== Cargando certificado ==' -ForegroundColor Cyan
$clave = Read-Host 'Clave del certificado' -AsSecureString
$cert = New-Object System.Security.Cryptography.X509Certificates.X509Certificate2(
    $Certificado, $clave,
    [System.Security.Cryptography.X509Certificates.X509KeyStorageFlags]::Exportable)
if (-not $cert.HasPrivateKey) { throw 'El .pfx no trae la clave privada: no sirve para firmar' }
Write-Host ("Firmante : {0}" -f $cert.Subject)
Write-Host ("Emisor   : {0}" -f $cert.Issuer)
Write-Host ("Vence    : {0:yyyy-MM-dd}" -f $cert.NotAfter)
if ($cert.Subject -eq $cert.Issuer) {
    Write-Host 'AVISO: es un certificado AUTOFIRMADO. Cada equipo debe confiar en él: hay que instalar su .cer en cada equipo.' -ForegroundColor Yellow
}

# 3. Firmar con sello de tiempo (la firma sigue siendo válida aunque el certificado venza)
Write-Host '== Firmando ==' -ForegroundColor Cyan
$r = Set-AuthenticodeSignature -FilePath $exe -Certificate $cert -HashAlgorithm SHA256 `
        -TimestampServer 'http://timestamp.digicert.com'
if (-not $r.SignerCertificate) { throw "No se pudo firmar: $($r.StatusMessage)" }

# 4. Comprobar
$v = Get-AuthenticodeSignature $exe
Write-Host ''
Write-Host ("Estado de la firma: {0}  ({1})" -f $v.Status, $v.StatusMessage)
if ($v.Status -eq 'Valid') {
    Write-Host 'Listo: el .exe quedó firmado y este equipo confía en la firma.' -ForegroundColor Green
} else {
    Write-Host 'El .exe quedó firmado, pero ESTE equipo todavía no confía en el certificado (normal si es autofirmado).' -ForegroundColor Yellow
}
