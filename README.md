# Sample ledger app using Encore, Temporal and TigerBeetle database

### API

1. `/authorize/:account_id/:amount`
   1. Starts a auth workflow
      1. Checks if the account exists and if the amount is available.
      2. Creates a pending transfer. This will reserve the funds for the transfer. TODO: Timeout is not working correctly
      3. Stores the transfer id in redis.
   2. Starts a void workflow
      1. Sleeps for the auth duration which is 10 seconds.
      2. Checks if the transfer is still pending. If it is, it will void the transfer.
2. `/present/:account_id/:amount`
   1. Starts a present workflow
      1. Checks if the account exists.
      2. Gets the transfer id from redis.
      3. Checks if the transfer is still pending. If it is, it will present the transfer.

### TODOS
1. Dockerize the app. Right now it is not possible to run the app without installing the dependencies.
2. Investigate more on TigerBeetle timeout.
3. More testing around timing issues. Right now there will be race conditions if the void and present workflows are started at the same time.
4. Add tests.
5. Start void as a child workflow instead of a separate workflow.
6. Move DB queries to a separate service, and probably implement Sagas to coordinate the workflows.

### How to run
1. Install all the dependencies. Encore, temporal-lite and TigerBeetle.
2. Start temporal-lite and TigerBeetle.
3. Start the app with `encore run --debug`
4. Create accounts using `account` API. Also create a main treasury account which is assumed here to be of id `1234567`.
5. Use `authorize` and `present` APIs to test the app.
