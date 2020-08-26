# -- FILE: features/keys.feature

Feature: Manage SSH keys
  SSH key files are managed and a manifest of their hashes are kept for easy searching.
    The primary key for identifying SSH key files is their "name".

    # -------------------------
    # Import
    Scenario: A pre-existing SSH key is added to hasp
    # Enter steps here

    # -------------------------
    # New
    Scenario: A new SSH key is requested to be generated
    # Enter steps here

    # -------------------------
    # Search
    Scenario: With a hash in hand, a user discovers if the key is known to hasp
    # Enter steps here

    Scenario: With a key file in hand, a user discovers if the key is known to hasp
    # Enter steps here

    # -------------------------
    # Maintenance
    Scenario: Duplicate keys are discovered and converted to symlinks
    # Enter steps here

    Scenario: Missing pub-keys are created
    # Enter steps here

    # -------------------------
    # Report
    Scenario: A single pane view is shown for all SSH keys
    # Enter steps here

    # -------------------------
    # Edit
    Scenario: A key is assigned to a host
    # Enter steps here

    Scenario: A key is assigned to an environment group
    # Enter steps here

    Scenario: A key is renamed (literal key remains the same)
    # Enter steps here

    Scenario: A literal key is replaced (name remains the same)
    # Enter steps here
