# Pocket Counter

A simple cli tool to count the articles in your getpocket.com list.

## Usage

First you need to go [here](https://getpocket.com/developer/apps/new) to create a new app and getting its `consumer_key`.

Fill out the form just like below.

![Pocket Developer Dashboard](/Pocket-Developer-Dashboard.png)

After creating the app, copy the `consumer_key`.

Then you need to install this project.

```bash
go get github.com/mehdy/Pocket-Counter
go install github.com/mehdy/Pocket-Counter
```

Run it using the `consumer_key` and follow the instructions.

```bash
Pocket-Counter -consumer-key 1234-abcd1234abcd1234abcd1234
```

And VOILA!!!

## Contribution

If anything goes wrong and the code doesn't work correctly please create an issue.

Any PRs are welcome!
