Level 5 notes (level05)
=======================

This level involves something called DomainAuthenticator, a quick, insecure way of authenticating yourself as a user from a domain.

Getting started
---------------

First, the easy thing to do is upload a authenticated.txt to the level02 server you were using before and then use that to login. The contents of the file should look like

```
AUTHENTICATED

```

with the trailing new line.

Then, you can submit the form to the DomainAuthenticator with the URL to that .txt file, and DomainAuthenticator will log you in!

What's the complication?
------------------------

To give you the key password you need, DomainAuthenicator requires that you've authenticated from the same domain it's running on - not level02.

What's the vulnerability?
-------------------------

The DomainAuthenticator authenticated page actually contains the text of your authenticated.txt file in it. You can use that to log you in!

Just submit your pingback URL as:

https://level05-_servernum_.stripe-ctf.com/user-_userstring_/?pingback=https%3A%2F%2Flevel02-_servernum_.stripe-ctf.com%2Fuser-_userstring_%2Fuploads%2Fauthenticated.txt

and you're logged in and can view the password.