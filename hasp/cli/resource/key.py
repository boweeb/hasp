"""Key commands"""

from pathlib import Path
import logging
from subprocess import run, CalledProcessError
from typing import Dict, List, Any

import click

from hasp.cli import option_, not_implemented
from hasp.io import write_ssh_key
from hasp.hasp import present_key_list, present_key, sync_keys

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
@option_("--batch", "-y", is_flag=True, default=False, help="No password (useful for batch testing)")
@click.pass_context
def new(ctx, key_type, name, comment, bits, batch):
    """Create a new SSH key."""
    ssh_dir: Path = ctx.obj["ssh_dir"]
    file_path: Path = ssh_dir / name
    type_: str = key_type.lower()
    type_map: Dict[str, str] = {
        "rsa": "rsa",
        "ec": "ed25519",
    }

    if ctx.parent.params["force"]:
        pub_file_path = Path(file_path.name + ".pub")
        file_path.unlink(missing_ok=True)
        pub_file_path.unlink(missing_ok=True)
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
    if batch:
        args += ["-N", ""]

    cmd = ["ssh-keygen", *args]
    LOG.info(f"{cmd=}")
    result = run(cmd, text=True, capture_output=True)
    try:
        result.check_returncode()
    except CalledProcessError:
        LOG.error("That did not go well. STDOUT:")
        LOG.error(result.stdout)
        raise click.Abort

    # Update state
    new_key_md = {"type": type_}
    if type_ == "rsa":
        new_key_md["bits"] = bits
    if comment:
        new_key_md["comment"] = comment

    # Commit to state file
    write_ssh_key(
        key_file_path=file_path,
        input_data=new_key_md,
    )


@click.command("key")
@click.pass_context
def edit(ctx):
    """Edit an existing SSH key."""
    not_implemented()  # TODO


@click.command("key")
@click.pass_context
def find(ctx):
    """Search for an SSH key."""
    not_implemented()  # TODO


@click.command("key")
@option_("-r", "--refresh", is_flag=True, help="TODO")
@click.pass_context
def list_(ctx, refresh):
    """List all known SSH keys."""
    present_key_list()


@click.command("key")
@option_("--name", "-n", type=click.STRING, required=True)
@click.pass_context
def show(ctx, name):
    """Show SSH key details."""
    present_key(name)


@click.command("key")
@option_("--all", "-a", "all_", is_flag=True, help="Sync all keys")
@option_("--name", "-n", help="SSH key to sync")
# @option_("--prune/--init", "-p/-i", "orientation", default=None, help="+-")
@option_("--prune", "-p", "orientation", flag_value="prune", help="Remove state entries with no real counterpart")
@option_("--init", "-i", "orientation", flag_value="init", help="Add missing state entries according to real files")
@option_("--both", "-b", "orientation", flag_value="both", default="both", help='(default) Both "prune" and "init"')
@click.pass_context
def sync(ctx, all_, name, orientation):
    """Synchronize SSH key state."""
    # TODO: Not thrilled with this.  There's no way to detect if multiple "orientation" flags were given.

    if all([all_, name]):
        raise click.UsageError("--all and --name are mutually exclusive.")

    sync_keys(name=name, orientation=orientation)
