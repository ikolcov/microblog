FROM ubuntu:22.04

RUN echo "Don't use cache #2" > /.force_full_rebuild

RUN apt-get update --fix-missing

RUN apt-get install -y \
    software-properties-common

RUN add-apt-repository -y \
    ppa:ubuntu-toolchain-r/test

RUN apt-get update

RUN apt-get install -y \
    git

RUN apt-get install -y \
    golang

WORKDIR /app

COPY . .

RUN go build -o /microblog

ENTRYPOINT ["/microblog"]
