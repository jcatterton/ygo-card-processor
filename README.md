##Yu-Gi-Oh Card Processor

Use tcgplayer.com API to collect card information from tcgplayer.com and process information in mongo database.

####Routes:
- GET /health: - Returns 200 if API is healthy and connected to database. Returns 500 if unable to connect to database.
- POST /process: - Updates all cards in database with values from tcgplayer.com API. Returns 200 once processing begins.
Returns 500 if error occurrs before processing begins. Processing continues after API sends response
- GET /card/{id} - Returns card from database based on given serial number. ID here does not refer to mongo objectID.
- POST /card/{id} - Adds card to database using information from tcgplayer.com API based on given serial number. ID here
does not refer to mongo objectID.
- PUT /card/{id} - Updates card in database using JSON values in request body based on given ID. ID here refers to mongo
ObjectID.
- DELETE /card/{id} - Deletes card from database based on given ID. ID here refers to mongo objectID.
- POST /cards - Adds card to database from excel file input. Excel file muse contain card serial number listed one by
one in first column of spreadsheet. Other columns are irrelevant. File must be given key 'input' in request. Returns 200
once adding has begun, 500 if error occurs before adding begins. Adding of cards continues after API has sent response.
- GET /cards - Returns all cards in database. Query parameters not supported at this time.


