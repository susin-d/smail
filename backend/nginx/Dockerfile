FROM nginx:1.25-alpine

RUN apk add --no-cache openssl bash
RUN rm -rf /var/cache/apk/*

# Copy configuration
COPY nginx.conf /etc/nginx/nginx.conf
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 80 443

CMD ["/entrypoint.sh"]
