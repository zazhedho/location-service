FROM nginx:1.29-alpine

WORKDIR /usr/share/nginx/html

COPY nginx.conf /etc/nginx/templates/default.conf.template
COPY index.html styles.css app.js ./

ENV API_BASE_URL=http://api:8080

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
