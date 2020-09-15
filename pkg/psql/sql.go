package psql

const (
	// Select item by id. Takes id as argument.
	Item = `
SELECT *
FROM items
WHERE id = $1`

	// Select items for one day (range). Takes a date as argument.
	ForDay = `
SELECT *
FROM items
WHERE created BETWEEN $1 AND $1::date+1`
)
