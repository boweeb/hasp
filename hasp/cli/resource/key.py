"""Key commands"""

from pathlib import Path
import logging
from subprocess import run, CalledProcessError
from typing import Dict, List

import click

from hasp.cli import option_, not_implemented


LOG = logging.getLogger(__name__)
LOG.debug("[key.py]")


@click.command("key")
@option_(
    "--type",
    "-t",
    "key_type",
    type=click.Choice(["RSA", "EC"], case_sensitive=False),
    required=False,
    default="EC",
    help="SSH key type.",
)
@option_("--name", "-n", type=click.STRING, required=True)
@option_("--comment", "-c", type=click.STRING)
@option_("--bits", "-b", type=click.Choice([str(i * 1024) for i in range(1, 5)]), default="4096")
@click.pass_context
def new(ctx, key_type, name, comment, bits):
    """Create a new SSH key."""
    # print(ctx.obj["ssh_dir"])
    # exit(0)

    state_file: Path = ctx.obj["state_file"]
    ssh_dir: Path = ctx.obj["ssh_dir"]
    file_path: Path = ssh_dir / name
    type_: str = key_type.lower()
    type_map: Dict[str, str] = {
        "rsa": "rsa",
        "ec": "ed25519",
    }

    if file_path.exists():
        LOG.error(f"{file_path} already exists!")
        raise click.Abort
    elif not ssh_dir.exists():
        LOG.error(f"{ssh_dir} does not exist!")
        raise click.Abort

    type_args = ["-t", type_map[type_]]
    file_args = ["-f", file_path.as_posix()]
    args: List[str] = [*type_args, *file_args]

    if comment:
        args += ["-C", comment]
    if type_ == "rsa":
        args += ["-b", bits]

    cmd = ["ssh-keygen", *args]
    LOG.info(f"{cmd=}")
    result = run(cmd, text=True, capture_output=True)
    try:
        result.check_returncode()
    except CalledProcessError:
        LOG.error("That did not go well. STDOUT:")
        LOG.error(result.stdout)
        raise click.Abort


@click.command("key")
@click.pass_context
def edit(ctx):
    """Edit an existing SSH key."""
    not_implemented()


@click.command("key")
@click.pass_context
def find(ctx):
    """Search for an SSH key."""
    not_implemented()


@click.command("key")
@click.pass_context
def list_(ctx):
    """List all known SSH keys."""
    not_implemented()


@click.command("key")
@click.pass_context
def show(ctx):
    """Show SSH key details."""
    not_implemented()
