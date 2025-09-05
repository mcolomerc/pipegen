-- Stream processing job - continuous insert
INSERT INTO revenue
SELECT name, amount as total
FROM transactions_v4;