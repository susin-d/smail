# Stage 1: Build Next.js Frontend
FROM node:20-alpine AS builder
WORKDIR /app
ARG NEXT_PUBLIC_API_URL=/api
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL

COPY frontend/package.json ./
RUN npm install --legacy-peer-deps

COPY frontend/ .
RUN npm run build

# Stage 2: Nginx Server
FROM nginx:1.25-alpine

RUN apk add --no-cache openssl bash
RUN rm -rf /var/cache/apk/*

# Copy configuration
COPY backend/nginx/nginx.conf /etc/nginx/nginx.conf
COPY backend/nginx/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Copy static frontend export
COPY --from=builder /app/out /usr/share/nginx/html

EXPOSE 80 443

CMD ["/entrypoint.sh"]
