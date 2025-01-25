# kwkhtmltopdf

A [wkhtmlpdf](https://wkhtmltopdf.org) server.

Why?

- avoid deploying wkhtmltopdf and it's dependencies in your application image
- keep the memory requirement of your application pods low while delegating
  memory hungry wkhtmltopdf jobs to dedicated pods
- easily select the wkhtmltopdf version to use at runtime

## WARNING

The server is not meant to be exposed to untrusted clients.

Several attack vectors exist (local file access being the most obvious).
Mitigating them is not a priority, since the main use case is
to use it as a private service.

## kwkhtmltopdf_server

A web server accepting [wkhtmlpdf](https://wkhtmltopdf.org) options and files
to convert as multipart form data.

It is written in go.

## Usage

cURL command - 
```curl
  curl --location 'http://localhost:8081/pdf' \
   --header 'X-Trace-ID: 123' \
   --form 'file=@"/Users/kshitizagrawal/Desktop/repositories/docker-wkhtmltopdf-aas/assets/header.html"' \
   --form 'file=@"/Users/kshitizagrawal/Desktop/repositories/docker-wkhtmltopdf-aas/assets/index.html"' \
   --form 'file=@"/Users/kshitizagrawal/Desktop/repositories/docker-wkhtmltopdf-aas/assets/footer.html"' \
   --form 'margin-top="20"' \
   --form 'page-size="A4"' \
   --form 'margin-bottom="10"' \
   --output "output.pdf"
```


## Quick start

### Run the server

```
$ docker compose up
```

The server should now listen on http://localhost:8080.

#### Note for Apple Silicon users

The docker image is built for amd64. If you are on Apple Silicon,
you can use it by disabling the `Use Rosetta for x86_64/amd64 emulation on Apple Silicon` option
in the Docker Desktop general settings first.

## Releasing

### Login to ECR

```sh
$ aws ecr get-login-password --region ap-south-1 | docker login --username AWS --password-stdin 909798297030.dkr.ecr.ap-south-1.amazonaws.com
```

### Build and push to ECR

```sh
$ docker buildx build -f Dockerfile-0.12.6.1 --platform linux/x86_64 --load --tag wkhtmltopdf-x86_64:0.0.16 .
$ docker tag wkhtmltopdf-x86_64:0.0.16 909798297030.dkr.ecr.ap-south-1.amazonaws.com/wkhtmltopdf-x86_64:0.0.16
$ docker push 909798297030.dkr.ecr.ap-south-1.amazonaws.com/wkhtmltopdf-x86_64:0.0.16
```

## Credits

Author: stephane.bidoul@acsone.eu.

Contributors are visible on
[GitHub](https://github.com/acsone/kwkhtmltopdf/graphs/contributors).

## License

MIT
