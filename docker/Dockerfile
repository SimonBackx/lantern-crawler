FROM ubuntu:16.04

# Install tor
COPY tor_apt_config.list /etc/apt/sources.list.d/

RUN gpg --keyserver keys.gnupg.net --recv A3C4F0F979CAA22CDBA8F512EE8CBC9E886DDD89
RUN gpg --export A3C4F0F979CAA22CDBA8F512EE8CBC9E886DDD89 | apt-key add -

# install tor and curl
RUN apt-get -qq update && apt-get install -qq tor deb.torproject.org-keyring curl -y

ENV GOPATH="/go"
ENV PATH="${PATH}:/usr/local/go/bin:${GOPATH}/bin"

# Install go
RUN curl -O https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz  \
    && (echo "53ab94104ee3923e228a2cb2116e5e462ad3ebaeea06ff04463479d7f12d27ca go1.8.linux-amd64.tar.gz" | sha256sum -c) \
    && tar xf go1.8.linux-amd64.tar.gz \
    && rm go1.8.linux-amd64.tar.gz \
    && chown -R root:root ./go \
    && mv go /usr/local \
    && echo "export GOPATH=/go \
export PATH=$PATH:/usr/local/go/bin:$GOPATH/bin" >> /etc/.profile \
    && mkdir -p /go/bin \
    && mkdir -p /go/pkg \
    && mkdir -p /go/src