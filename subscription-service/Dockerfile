FROM node:lts-slim
LABEL authors="waraidako"

RUN apt-get update && apt-get install -y --no-install-recommends \
    cron \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /subscription-service/src

COPY package*.json ./
RUN npm ci

COPY . .

COPY crontab /etc/cron.d/scraper
RUN chmod 0644 /etc/cron.d/subscription-service

CMD ["cron", "-f"]