param(
    [string]$EditorPath = 'C:\Program Files\Unity\Hub\Editor\6000.6.7f1\Editor\Unity.exe',
    [string]$BackendUrl = 'http://192.168.1.100:8080',
    [string]$LocalMavenCache = ''
)
$ErrorActionPreference = 'Stop'
if (-not (Test-Path -LiteralPath $EditorPath)) { throw 'Unity Editor not found; set -EditorPath.' }
$projectPath = Join-Path $PSScriptRoot 'unity'
$logPath = Join-Path ([System.IO.Path]::GetTempPath()) 'inspection-android-build.log'
$oldJavaOptions = $env:JAVA_TOOL_OPTIONS
$oldBackend = $env:INSPECTION_ANDROID_BACKEND
$oldMaven = $env:INSPECTION_LOCAL_MAVEN
try {
    $env:INSPECTION_ANDROID_BACKEND = $BackendUrl
    $env:INSPECTION_LOCAL_MAVEN = $LocalMavenCache
    $destination = [uri]'https://maven.aliyun.com/repository/google/'
    $proxy = [System.Net.WebRequest]::DefaultWebProxy.GetProxy($destination)
    if ($proxy -and $proxy.AbsoluteUri -ne $destination.AbsoluteUri -and [string]::IsNullOrEmpty($proxy.UserInfo)) {
        $env:JAVA_TOOL_OPTIONS = "$oldJavaOptions -Dhttp.proxyHost=$($proxy.Host) -Dhttp.proxyPort=$($proxy.Port) -Dhttps.proxyHost=$($proxy.Host) -Dhttps.proxyPort=$($proxy.Port) -Dhttp.nonProxyHosts=localhost|127.0.0.1 -Dorg.gradle.daemon=false"
    }
    $arguments = @('-batchmode','-nographics','-quit','-buildTarget','Android','-projectPath',('"'+$projectPath+'"'),'-executeMethod','ProjectSetup.BuildAndroid','-logFile',('"'+$logPath+'"'))
    $process = Start-Process -FilePath $EditorPath -ArgumentList $arguments -WindowStyle Hidden -Wait -PassThru
    if ($process.ExitCode -ne 0 -or -not (Select-String -LiteralPath $logPath -SimpleMatch 'INSPECTION_ANDROID_BUILD_PASS' -Quiet)) { throw "Android build failed. Local log: $logPath" }
    Write-Output (Join-Path $PSScriptRoot 'android\Inspection-Android.apk')
} finally {
    $env:JAVA_TOOL_OPTIONS = $oldJavaOptions
    $env:INSPECTION_ANDROID_BACKEND = $oldBackend
    $env:INSPECTION_LOCAL_MAVEN = $oldMaven
}
