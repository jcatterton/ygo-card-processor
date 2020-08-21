##Yu-Gi-Oh Card Processor

Fetches the HTML from yugiohprices.com for given cards and parses the HTML to get
the current prices for that card. Outputs the prices into a spreadsheet with the
card name, highest current price, lowest current price, and average current price.

###How to use

Save a spreadsheet in the project's root directory called "cardlist.xlsx". This card
should consist of a single column. Each cell in the column should have one serial number
(Serial-numbers are found below the bottom right corner of the card's picture, and follow
the format XXXX-XXXX or similar). Add as many serial numbers as you wish to track.
Serial numbers are not case sensitive. The application will ignore any duplicate serials.

Next, run the project with `go run main/main.go`, the console will show progress for
fetching card information. The console will alert you if an serial number could not
be found, which indicates invalid entry.

Each card typically takes less than a second to process, but this means large card
collections may take several minutes.

After running, a new spreadsheet called "output.xlsx" will be in the project's root
directory containing price information on all provided cards. Prices are separated
into three columns, minimum, maximum, and average. Totals for the three columns are
at the bottom.
