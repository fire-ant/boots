local_resource(
  'compile boots',
  'make cmd/boots/boots-linux-amd64'
)
docker_build(
    'boots_boots',
    '.',
    dockerfile='Dockerfile',
    only=['./cmd/boots']
)
docker_compose(['./docker-compose.yml'])
