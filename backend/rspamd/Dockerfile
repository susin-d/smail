FROM alpine:3.19

RUN set -x \
    && apk add --no-cache rspamd rspamd-client rspamd-controller rspamd-proxy ca-certificates tzdata

# Create directory for custom configs
RUN mkdir -p /etc/rspamd/local.d /var/lib/rspamd

# Set ownership
RUN chown -R rspamd:rspamd /var/lib/rspamd /etc/rspamd

USER rspamd

EXPOSE 11332 11333 11334

CMD ["rspamd", "-i", "-f"]
