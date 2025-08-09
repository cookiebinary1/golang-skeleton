# syntax=docker/dockerfile:1
FROM node:24-alpine AS dependencies
WORKDIR /app
COPY services/web/package.json services/web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci

FROM dependencies AS dev
COPY services/web ./
EXPOSE 3000
CMD ["npm", "run", "dev"]

FROM dependencies AS build
COPY services/web ./
RUN npm run build

FROM nginx:stable-alpine AS release
COPY services/web/nginx.conf /etc/nginx/nginx.conf
COPY --from=build /app/dist /usr/share/nginx/html
USER nginx
EXPOSE 8080
ENTRYPOINT ["nginx", "-g", "daemon off;"]
