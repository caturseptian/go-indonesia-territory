# Uji integrasi

Modul terpisah agar modul utama tetap tanpa dependensi. SQLite selalu diuji; Postgres/MySQL/MariaDB bila DSN diisi:

    TERRITORY_PG_DSN='postgres:///territory_test?host=/tmp' \
    TERRITORY_MYSQL_DSN='root@tcp(127.0.0.1:3306)/territory_test' \
    TERRITORY_MARIADB_DSN='root@tcp(127.0.0.1:3307)/territory_test' \
    go test -race ./...
