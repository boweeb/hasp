"""Compose "edit" command subgroup."""

import logging

import click

from hasp.cli.resource.key import edit as key
from hasp.cli.resource.host import edit as host
from hasp.cli.resource.config import edit as config


LOG = logging.getLogger(__name__)
LOG.debug("[edit.py]")


@click.group("edit")
@click.pass_context
def group(
    ctx,
):
    """Edit a resource."""
    LOG.debug("[EDIT]")


group.add_command(key)
group.add_command(host)
group.add_command(config)
