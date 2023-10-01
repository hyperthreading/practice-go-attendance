FROM debian:bookworm

RUN apt update
RUN apt install -y curl

ENTRYPOINT [ "/main" ]