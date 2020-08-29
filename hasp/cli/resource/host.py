"""Host commands"""

import logging

import click

from hasp.cli import option_, not_implemented


LOG = logging.getLogger(__name__)
LOG.debug("[host.py]")


@click.command("host")
@click.pass_context
def new(ctx):
    """."""
    not_implemented()


@click.command("host")
@click.pass_context
def edit(ctx):
    """."""
    not_implemented()


@click.command("host")
@click.pass_context
def find(ctx):
    """."""
    not_implemented()


@click.command("host")
@click.pass_context
def list_(ctx):
    """."""
    not_implemented()


@click.command("host")
@click.pass_context
def show(ctx):
    """."""
    not_implemented()
