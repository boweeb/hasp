"""Host commands"""

import logging

import click

from hasp.cli import option_


LOG = logging.getLogger(__name__)
LOG.debug("[host.py]")


@click.command("host")
@option_("--verbose", "-v", is_flag=True, required=False, help="Enable debug-level logging")
@click.pass_context
def new(ctx, verbose):
    """foobar"""
    print("asdf")
    print(verbose)


@click.command("host")
@option_("--verbose", "-v", is_flag=True, required=False, help="Enable debug-level logging")
@click.pass_context
def edit(ctx, verbose):
    """foobar"""
    print("asdf")
    print(verbose)


@click.command("host")
@option_("--verbose", "-v", is_flag=True, required=False, help="Enable debug-level logging")
@click.pass_context
def find(ctx, verbose):
    """foobar"""
    print("asdf")
    print(verbose)


@click.command("host")
@option_("--verbose", "-v", is_flag=True, required=False, help="Enable debug-level logging")
@click.pass_context
def list_(ctx, verbose):
    """foobar"""
    print("asdf")
    print(verbose)


@click.command("host")
@option_("--verbose", "-v", is_flag=True, required=False, help="Enable debug-level logging")
@click.pass_context
def show(ctx, verbose):
    """foobar"""
    print("asdf")
    print(verbose)
