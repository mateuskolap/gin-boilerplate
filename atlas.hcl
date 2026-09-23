variable "database_url" {
  type    = string
  default = getenv("DATABASE_URL")
}

env "local" {
  url = var.database_url
  dev = "docker://postgres/18/dev?search_path=public"

  migration {
    dir = "file://migrations"
  }
}
