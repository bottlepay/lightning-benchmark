storage "s3" {
  access_key = "ROOTUSER"
  secret_key = "CHANGEME123"
  endpoint = "http://minio:9000"
  bucket = "vault-storage"
  s3_force_path_style = "true"
}

listener "tcp" {
  address     = "0.0.0.0:8200"
  tls_disable = "true"
}

api_addr = "http://127.0.0.1:8200"
#cluster_addr = "https://127.0.0.1:8201"

ui = true

# Build plugin with GOOS=linux
plugin_directory = "/vault/plugins"
log-level = "trace"
disable_mlock = true