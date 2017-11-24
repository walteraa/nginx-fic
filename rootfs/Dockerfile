FROM gcr.io/google_containers/ubuntu-slim:0.14

RUN apt-get update && apt-get install -y --no-install-recommends \
      bash
COPY federation-nginx-controller . 
COPY nginx.tmpl .
