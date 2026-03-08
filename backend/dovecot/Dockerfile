FROM alpine:3.19

RUN apk add --no-cache \
    dovecot \
    dovecot-lmtpd \
    dovecot-sql \
    dovecot-mysql \
    dovecot-pigeonhole-plugin \
    bash \
    ca-certificates

RUN rm -rf /var/cache/apk/*

COPY dovecot.conf /etc/dovecot/dovecot.conf
COPY dovecot-sql.conf /etc/dovecot/dovecot-sql.conf
COPY sieve/ /etc/dovecot/sieve/
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 993 24 12345

ENTRYPOINT ["/entrypoint.sh"]
