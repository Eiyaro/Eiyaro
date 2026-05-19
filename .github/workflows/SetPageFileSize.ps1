# Increase pagefile size on Windows to avoid running out of memory during tests
# This script is called from GitHub Actions

$currentPageFileSize = (Get-CimInstance Win32_PageFileSetting).InitialSize
Write-Host "Current pagefile size: $currentPageFileSize MB"

# Set pagefile to 8GB if it's less than that
if ($currentPageFileSize -lt 8192) {
    Write-Host "Increasing pagefile size to 8GB..."
    $pageFile = Get-CimInstance Win32_PageFileSetting
    $pageFile.InitialSize = 8192
    $pageFile.MaximumSize = 8192
    $pageFile.Put()
    Write-Host "Pagefile size increased successfully"
} else {
    Write-Host "Pagefile size is sufficient"
}
