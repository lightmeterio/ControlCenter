# Registration

Test the administrator registration process

## Registration fails
Tags: fails

* Go to registration page
* Focus on field with placeholder "Name"
* Type "User Complete Name"
* Focus on field with placeholder "Email"
* Type "user@example.com"
* Focus on field with placeholder "Password"
* Type "54353%#%#54354353gffgdgdfg"
* Expect registration to fail
* Click on "Register"

## Registraton succeeds
Tags: succeeds

* Go to registration page
* Focus on field with placeholder "Name"
* Type "User Complete Name"
* Focus on field with placeholder "Email"
* Type "user@example.com"
* Focus on field with placeholder "Password"
* Type "54353%#%#54354353gffgdgdfg"
* Select "Most of my mail is…" from menu "direct"
* Expect registration to fail
* Click on "Register"