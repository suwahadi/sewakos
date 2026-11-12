FROM node:22-bookworm-slim AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG STORAGE_PUBLIC_BASE_URL
ARG NEXT_PUBLIC_MIDTRANS_CLIENT_KEY=""
ARG NEXT_PUBLIC_MIDTRANS_IS_PRODUCTION=false
ARG NEXT_PUBLIC_APP_NAME="Sewa Kos"
ENV NEXT_TELEMETRY_DISABLED=1 \
    STORAGE_PUBLIC_BASE_URL=$STORAGE_PUBLIC_BASE_URL \
    NEXT_PUBLIC_MIDTRANS_CLIENT_KEY=$NEXT_PUBLIC_MIDTRANS_CLIENT_KEY \
    NEXT_PUBLIC_MIDTRANS_IS_PRODUCTION=$NEXT_PUBLIC_MIDTRANS_IS_PRODUCTION \
    NEXT_PUBLIC_APP_NAME=$NEXT_PUBLIC_APP_NAME
RUN npm run build && npm prune --omit=dev

FROM node:22-bookworm-slim
WORKDIR /app
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1 PORT=3000 HOSTNAME=0.0.0.0
COPY --from=build --chown=node:node /app/package.json /app/package-lock.json /app/next.config.ts ./
COPY --from=build --chown=node:node /app/node_modules ./node_modules
COPY --from=build --chown=node:node /app/.next ./.next
USER node
EXPOSE 3000
CMD ["node_modules/.bin/next", "start"]
