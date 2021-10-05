FROM node:14.18

COPY about.html /www/about.html
COPY index.html /www/index.html

RUN npm install -g serve

CMD ["serve", "www"]