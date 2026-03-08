FROM alpine:3.19

RUN apk add --no-cache \
    postfix \
    postfix-pcre \
    openssl \
    ca-certificates \
    bash

# Clean up
RUN rm -rf /var/cache/apk/*

COPY main.cf /etc/postfix/main.cf
COPY master.cf /etc/postfix/master.cf
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 25 587

ENTRYPOINT ["/entrypoint.sh"]
