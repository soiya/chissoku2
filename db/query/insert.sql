-- name: InsertData :exec
INSERT INTO co2 (co2, humidity, temperature, timestamp) VALUES ($1, $2, $3, $4);
