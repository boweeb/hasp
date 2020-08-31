from typing import List, Text, Any, Optional, Mapping, Union
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
from tomlkit import table as toml_table


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

    init = False
    if orientation in ["init", "both", None]:
        init = True

    prune = False
    if orientation in ["prune", "both", None]:
        prune = True

    if init:
        LOG.info("Begin SSH key initialization")
        if name:
            init_state_entries(c.ssh_dir / name)
        else:
            for key_file in c.ssh_dir.glob("*"):
                init_state_entries(key_file)

    if prune:
        LOG.info("Begin SSH key pruning")
        if name:
            prune_state_entries(name)
        else:
            for key in resolve_toml_keys(c.state.toml_doc["ssh_key"]):
                prune_state_entries(key)


def prune_state_entries(key_name: Text):
    c = ContextBorg()
    key_map: toml_table = c.state.toml_doc["ssh_key"]

    resolved_key_list = resolve_toml_keys(key_map)

    if key_name in resolved_key_list:
        if not Path(c.ssh_dir / key_name).exists():
            LOG.info(f'Pruning stale SSH key entry in state: "{key_name}"')
            key_map.remove(key_name)
            # commit_toml()
            c.state.toml_file.write(c.state.toml_doc)
    else:
        LOG.info(f'Key does is not in state: "{key_name}"')


def resolve_toml_keys(element) -> List[Text]:
    # Resolve generator into actual list
    resolved_list = []
    for k in element.keys():
        resolved_list.append(k)
    return resolved_list


def read_real_key(key_file: Path):
    # c = ContextBorg()

    magic_type = magic.from_file(key_file.as_posix())
    if re.match(r"(?i)^(OpenSSH|PEM).*private.*\bkey$", magic_type):
        LOG.info(f'Found private key: "{key_file.name}"')
        cmd = ["ssh-keygen", "-l", "-E", "md5", "-f", key_file]
        result = run(cmd, text=True, capture_output=True, check=True)
        lines = result.stdout.rstrip().split()

        bits = int(lines.pop(0))
        hash_base = lines.pop(0).split(":")
        _hash_md5 = hash_base.pop(0)
        hash_value = ":".join(hash_base)
        type_ = lines.pop(-1).lstrip('(').rstrip(')').lower()
        comment = " ".join(lines)

        real_key = {
            "type": type_,
            "comment": comment,
            "bits": bits,
            "md5": hash_value,
            "format": magic_type.split(" ")[0].lower(),
        }

        return real_key


def init_state_entries(key_file):
    # c = ContextBorg()

    if key_file.is_file() and not key_file.is_symlink():
        real_key = read_real_key(key_file)
        update_key(key_file, real_key)
    elif key_file.is_file() and key_file.is_symlink():
        target = key_file.resolve()
        LOG.info(f'Found symlink: "{key_file.name}" --> "{target.name}"')
        update_key(target, aliases=[key_file])
    else:
        LOG.warning(f'SKIPPING: File does not exist OR is a symlink: "{key_file}"')
        # raise click.Abort


def update_key(
    key_file: Path,
    data: Optional[Mapping[Text, Union[Text, int]]] = None,
    aliases: Optional[List[Path]] = None
):
    c = ContextBorg()

    key_map = c.state.toml_doc["ssh_key"]
    resolved_key_list = resolve_toml_keys(key_map)

    if key_file.name in resolved_key_list:
        state_key = key_map[key_file.name]

        LOG.debug(f"{state_key=}")
        LOG.debug(f"{data=}")

        if data:
            for attr in ["type", "bits", "comment", "md5", "format"]:

                if attr not in state_key.keys():
                    print(f"{attr:<10}:            <NEW> --> {data[attr]}")
                elif state_key[attr] != data[attr]:
                    print(f'{attr:<10}: {state_key[attr]:>16} --> {data[attr]}')

                state_key[attr] = data[attr]

        if aliases:
            resolved_list = resolve_toml_keys(state_key)

            for alias in aliases:
                if "aliases" not in resolved_list:
                    state_key["aliases"] = []

                alias_set = set(state_key["aliases"])
                alias_set.add(alias.name)
                state_key["aliases"] = list(alias_set)

    else:
        # "if aliases" does not apply here.
        if data:
            LOG.info("Importing new SSH key entry into state")
            key_map[key_file.name] = deepcopy(data)

    c.state.toml_file.write(c.state.toml_doc)
    # commit_toml()


def commit_toml() -> None:
    c = ContextBorg()
    c.state.toml_file.write(c.state.toml_doc)
