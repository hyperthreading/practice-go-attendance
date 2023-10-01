FROM golang:1.21-bookworm AS builder

WORKDIR /app

RUN apt update
RUN apt install -y inotify-tools git

RUN git config --global --add safe.directory /app 

CMD ["bash", "./build/test/build_watch.sh"]