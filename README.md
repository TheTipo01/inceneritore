# inceneritore

[![Go Report Card](https://goreportcard.com/badge/github.com/TheTipo01/inceneritore)](https://goreportcard.com/report/github.com/TheTipo01/inceneritore)

Discord bot for kicking people, reinviting them, and tracking all of that.

Remember to modify the provided `example_config.yml`, adding you own credentials and renaming it to `config.yml`

To add a server, add a record to the config table of the database (remember to start the bot at least once to generate
the tables).

Explanation of the columns in the table:

* `serverId` is the id of the server
* `serverName` the name of the server
* `ruolo` the role to give to the user, so they can't do anything in the server while in the voice channel
* `testuale` the channel of the server where to send when someone gets incinerated
* `vocale` the voice channel where people get kicked
* `invito` the invite for the server.
