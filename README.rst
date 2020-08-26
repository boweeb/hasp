hasp
====

``hasp`` is all about managing *identity*.  Initially, this concerned with SSH.
Therefore, ``hasp`` strives to manage SSH keys and configuration.
See [Future Ideas] -- someday Git and GPG identities may be added to ``hasp``-management as well.


CLI Examples
------------


.. code-block:: bash

    # Where
    #     "object" is one of: "key", "host", "config", etc.
    #     "verb" is one of: "new", "edit", "show", "list", etc.

    hasp <verb> <object> [options] [arguments]

Keys

.. code-block:: bash

    hasp new key --name "laptop"
    hasp edit key --name "laptop" --comment "Local mac laptop"
    hasp show key --name "laptop"
    hasp find key --md5 "de:ad:be:ef"
                  --md5 "deadbeef"
                  --md5 "DEADBEEF"
    hasp list keys
    hasp list keys --match "lap"  # Simple "contians" match on "name" property

Hosts

.. code-block:: bash

    hasp new host --name "laptop"
    hasp edit host
    hasp show host
    hasp find host
    hasp list hosts

Config

.. code-block:: bash

    hasp new config --name "laptop"
    hasp edit config
    hasp show config
    # hasp find config  # useful?  I'm not sure I'll implement this.
    hasp list configs


Primary Goals
-------------

* Manage SSH keys
    - Support aliases
* Manage SSH host config
    - Preserve comments, ordering, etc.
    - Support comment decoration
* SSH config file may be edited by hand safely (won't lose edits)
* Updates create rolling backups
* Leverage profiles and profile groups
    - ie. Groups for "personal" and "work"
    - Multiple profiles are supported for each group
        Honestly, I personally don't have a current need for multiple profiles per group
        but I want to plan ahead for this flexibility.


Future Ideas
------------

* Manage GPG (personal id) keys
    - Does **not** replace keyrings!
    - The purpose is two-fold:
        1. Dead-simple overlay for import
        2. Make ``hasp`` *aware* of GPG keys for the sake of Git config (for signing commits)
* Manage Git config
    - Global and project-local
    - For managing "identity"
        + Config keys
            * ``user.name``
            * ``user.email``
            * ``user.signingkey``
        + Perhaps store the above in a "profile"


Reference
---------

Glean lessons-learned from:
* https://github.com/emre/storm
* ``advanced-ssh-config`` (at ``./vendor/advanced-ssh-config``)
