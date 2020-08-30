from typing import List, Text, Any, Optional
from hasp.io import ContextBorg, get_key_list
from rich import print as rprint
from pprint import pprint
from copy import deepcopy
import logging
from pathlib import Path
import magic
import re
import click
from subprocess import run, CalledProcessError


LOG = logging.getLogger(__name__)


def present_key_list() -> None:
    c = ContextBorg()
    key_list: List[Any] = c.state.toml_doc["ssh_key"].keys()
    for key in key_list:
        present_key(key)


def present_key(key) -> None:
    c = ContextBorg()

    key_map = {}
    for k, v in c.state.toml_doc["ssh_key"][key].items():
        key_map[k] = v

    key_copy = deepcopy(key_map)

    print(key)

    # print("\t", end="")
    # pprint(c.state.toml_doc["ssh_key"][key])
    # rprint(key_copy)
    for k, v in key_copy.items():
        print(f'\t{k:>8}: "{v}"')
    print()


def sync_keys(name: Optional[Text], orientation: Optional[Text]) -> None:
    LOG.debug(f"{name=}")
    LOG.debug(f"{orientation=}")

    c = ContextBorg()

    if name:
        key_file = Path(c.ssh_dir / name)
        if key_file.is_file() and not key_file.is_symlink():
            magic_type = magic.from_file(key_file.as_posix())
            if re.match(r"^(OpenSSH|PEM).*\bkey$", magic_type):
                LOG.info(f'Found private key: "{key_file.name}"')
                cmd = ["ssh-keygen", "-l", "-E", "md5", "-f", key_file]
                result = run(cmd, text=True, capture_output=True, check=True)
                lines = result.stdout.rstrip().split()

                bits = int(lines.pop(0))
                hash_base = lines.pop(0).split(":")
                hash_md5 = hash_base.pop(0)
                hash_value = ":".join(hash_base)
                type_ = lines.pop(-1).lstrip('(').rstrip(')')
                comment = " ".join(lines)

                real_key = {
                    "type": type_,
                    "comment": comment,
                    "bits": bits,
                }

                update_key(name, real_key)
        else:
            LOG.error(f'File does not exist: "{key_file}"')
            raise click.Abort
    else:
        all_files = {}
        for key_file in c.ssh_dir.glob("*"):
            if key_file.is_file() and not key_file.is_symlink():
                magic_type = magic.from_file(key_file.as_posix())
                if re.match(r"^(OpenSSH|PEM).*\bkey$", magic_type):
                    LOG.info(f'Found private key: "{key_file.name}"')
                # print(f"{g.name} -- {magic.from_file(g.as_posix())}")

    print('---')


def update_key(name, data):
    c = ContextBorg()

    key_map = c.state.toml_doc["ssh_key"]
    if name not in key_map.keys():
        print("uh oh")

    state_key = key_map[name]

    LOG.debug(f"{state_key=}")
    LOG.debug(f"{data=}")

    for attr in ["type", "bits", "comment"]:
        if attr not in state_key.keys():
            print(f"{attr:<10}:            <NEW> --> {data[attr]}")

        else:
            print(f'{attr:<10}: {state_key[attr]:>16} --> {data[attr]}')

        state_key[attr] = data[attr]
        # state_key["hashes"] = {"md5": "asdf", "sha256": "qwer"}
        # state_key["hashes"] = [{"type": "md5", "value": "asdf"}, {"type": "sha256", "value": "qwer"}]
        state_key["md5"] = "qwer"
        c.state.toml_file.write(c.state.toml_doc)
