$templatePath = "C:\Users\Neo\Desktop\WWW\go\services_obywatel_app\platform\templates\service-template"

Write-Host "Creating Go microservice template..." -ForegroundColor Green

# Root files
$files = @(
    ".env.example",
    ".gitignore",
    "Dockerfile",
    "README.md",
    "go.mod",
    "go.sum",

    # cmd
    "cmd\main.go",

    # config
    "config\config.go",
    "config\app.go",
    "config\db.go",

    # internal
    "internal\di\container.go",

    "internal\handler\.gitkeep",

    "internal\model\.gitkeep",

    "internal\service\.gitkeep",

    "internal\repository\.gitkeep",

    "internal\router\routes.go",
    "internal\router\health.go",

    "internal\middleware\.gitkeep",

    # database
    "db\schema.sql",
    "db\query.sql",

    # sqlc
    "sqlc.yaml"
)

foreach ($file in $files) {

    $fullPath = Join-Path $templatePath $file
    $directory = Split-Path $fullPath

    if (!(Test-Path $directory)) {
        New-Item -ItemType Directory -Path $directory -Force | Out-Null
    }

    if (!(Test-Path $fullPath)) {
        New-Item -ItemType File -Path $fullPath -Force | Out-Null
    }

    Write-Host "Created: $file"
}


# Git init
Set-Location $templatePath

if (!(Test-Path ".git")) {
    git init
    Write-Host "Git initialized" -ForegroundColor Cyan
}
else {
    Write-Host "Git already exists" -ForegroundColor Yellow
}


# Initial commit
git add .

git commit -m "Initial microservice template structure"

Write-Host ""
Write-Host "Template created successfully!" -ForegroundColor Green
Write-Host $templatePath