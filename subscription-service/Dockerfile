FROM node:lts-slim
LABEL authors="waraidako"

RUN apt-get update && apt-get install -y --no-install-recommends \
    cron \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /subscription-service

COPY subscription-service/package*.json ./
RUN npm ci

COPY subscription-service/ .
copy proto/ /proto/

COPY subscription-service/crontab /etc/cron.d/subscription-service
RUN chmod 0644 /etc/cron.d/subscription-service

CMD ["cron", "-f"]