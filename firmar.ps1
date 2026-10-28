# Compila, firma organizador_carpetas.exe y genera el instalador del certificado para los usuarios.
#
# Uso (en PowerShell, dentro de la carpeta del proyecto):
#   .\firmar.ps1                 -> compila con wails, firma y genera el paquete
#   .\firmar.ps1 -SinCompilar    -> solo firma el .exe que ya existe y genera el paquete
#   .\firmar.ps1 -Huella <thumbprint>  -> usa un certificado concreto
#
# Resultado en build\bin\ (lo que se le entrega al usuario):
#   organizador_carpetas.exe     (firmado)
#   instalar_certificado.cmd     (instala la parte pública del certificado; la trae adentro; no usa PowerShell)
#   S4M-Certificate.cer          (la parte pública sola, para instalarla a mano o por GPO)
#
# Si PowerShell no deja ejecutar scripts:  powershell -ExecutionPolicy Bypass -File .\firmar.ps1

param(
    [string]$Huella,
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

$bin = Join-Path $PSScriptRoot 'build\bin'
$exe = Join-Path $bin 'organizador_carpetas.exe'
if (-not (Test-Path $exe)) { throw "No se encontró $exe" }

# 2. Certificado: primero el almacén de Windows, si no el .pfx
Write-Host '== Buscando certificado ==' -ForegroundColor Cyan
$cert = $null
$almacen = Get-ChildItem Cert:\CurrentUser\My -CodeSigningCert | Where-Object HasPrivateKey
if ($Huella) {
    $cert = $almacen | Where-Object Thumbprint -eq $Huella | Select-Object -First 1
    if (-not $cert) { throw "No hay certificado de firma con huella $Huella en Cert:\CurrentUser\My" }
} else {
    $cert = $almacen | Where-Object { $_.Subject -like '*S4M*' -and $_.NotAfter -gt (Get-Date) } |
            Sort-Object NotAfter -Descending | Select-Object -First 1
}
if (-not $cert) {
    if (-not (Test-Path $Certificado)) { throw "No hay certificado en el almacén ni en $Certificado" }
    Write-Host "Usando $Certificado (Enter si no tiene clave)"
    $clave = Read-Host 'Clave del certificado' -AsSecureString
    $cert = New-Object System.Security.Cryptography.X509Certificates.X509Certificate2(
        $Certificado, $clave,
        [System.Security.Cryptography.X509Certificates.X509KeyStorageFlags]::Exportable)
    if (-not $cert.HasPrivateKey) { throw 'El .pfx no trae la clave privada: no sirve para firmar' }
}
Write-Host ("Firmante : {0}" -f $cert.Subject)
Write-Host ("Huella   : {0}" -f $cert.Thumbprint)
Write-Host ("Vence    : {0:yyyy-MM-dd}" -f $cert.NotAfter)

# 3. Firmar con sello de tiempo (la firma sigue siendo válida aunque el certificado venza)
Write-Host '== Firmando ==' -ForegroundColor Cyan
$r = Set-AuthenticodeSignature -FilePath $exe -Certificate $cert -HashAlgorithm SHA256 `
        -TimestampServer 'http://timestamp.digicert.com'
if (-not $r.SignerCertificate) { throw "No se pudo firmar: $($r.StatusMessage)" }

# 4. Instalador del certificado para los usuarios: un .cmd (no depende de PowerShell, que suele estar
#    bloqueado en los equipos) con la parte PÚBLICA del certificado adentro. También se deja el .cer
#    por si se prefiere instalarlo a mano (doble clic -> Instalar certificado).
Write-Host '== Generando instalar_certificado.cmd y S4M-Certificate.cer ==' -ForegroundColor Cyan
$der = $cert.Export([System.Security.Cryptography.X509Certificates.X509ContentType]::Cert)
[IO.File]::WriteAllBytes((Join-Path $bin 'S4M-Certificate.cer'), $der)
$b64 = [Convert]::ToBase64String($der)
$lineas = for ($k = 0; $k -lt $b64.Length; $k += 64) { '>>"%CER%" echo ' + $b64.Substring($k, [Math]::Min(64, $b64.Length - $k)) }
Remove-Item (Join-Path $bin 'instalar_certificado.ps1') -ErrorAction SilentlyContinue
$cmd = @"
@echo off
setlocal
chcp 65001 >nul
title Instalar certificado de organizador_carpetas
rem Instala la parte publica del certificado de S4M para que Windows confie en organizador_carpetas.exe.
rem Se ejecuta UNA vez por equipo (doble clic). No usa PowerShell. No permite firmar nada.
echo.
echo  Instalando el certificado de S4M (huella $($cert.Thumbprint))...
echo.
set "CER=%TEMP%\S4M-Certificate.cer"
>"%CER%" echo -----BEGIN CERTIFICATE-----
$($lineas -join "`r`n")
>>"%CER%" echo -----END CERTIFICATE-----

rem Con permisos de administrador se instala para todo el equipo; sin ellos, solo para este usuario.
net session >nul 2>&1
if %errorlevel%==0 (
  certutil -f -addstore Root "%CER%" >nul && certutil -f -addstore TrustedPublisher "%CER%" >nul
) else (
  echo  Sin permisos de administrador: se instala solo para su usuario.
  echo  Windows mostrara una ADVERTENCIA DE SEGURIDAD: responda "Si".
  echo.
  certutil -f -user -addstore Root "%CER%" >nul && certutil -f -user -addstore TrustedPublisher "%CER%" >nul
)
set "R=%errorlevel%"
del "%CER%" >nul 2>&1

rem Quita la marca de "descargado de Internet" del .exe que este en la misma carpeta.
if exist "%~dp0organizador_carpetas.exe" type nul > "%~dp0organizador_carpetas.exe:Zone.Identifier" 2>nul

echo.
if "%R%"=="0" (
  echo  Listo: el certificado quedo instalado. Ya puede abrir organizador_carpetas.exe.
) else (
  echo  No se pudo instalar el certificado. Pruebe: clic derecho en este archivo, "Ejecutar como administrador".
)
echo.
pause
"@
[IO.File]::WriteAllText((Join-Path $bin 'instalar_certificado.cmd'), $cmd.Replace("`r`n", "`n").Replace("`n", "`r`n"), (New-Object System.Text.UTF8Encoding($false)))

# 5. Comprobar
$v = Get-AuthenticodeSignature $exe
Write-Host ''
Write-Host ("Estado de la firma en este equipo: {0}" -f $v.Status)
Write-Host ''
Write-Host 'Entregar al usuario (carpeta build\bin):' -ForegroundColor Green
Write-Host '  organizador_carpetas.exe'
Write-Host '  instalar_certificado.cmd   (una vez por equipo: doble clic; no usa PowerShell)'
Write-Host '  S4M-Certificate.cer        (opcional: para instalarlo a mano o que TI lo distribuya)'
