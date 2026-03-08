FROM alpine:3.19

RUN apk add --no-cache \
    opendkim \
    opendkim-utils \
    bash

RUN rm -rf /var/cache/apk/*

COPY opendkim.conf /etc/opendkim/opendkim.conf
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

RUN mkdir -p /etc/opendkim/keys /run/opendkim
RUN chown -R opendkim:opendkim /etc/opendkim /run/opendkim

EXPOSE 8891

ENTRYPOINT ["/entrypoint.sh"]
