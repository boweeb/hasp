from typing import List, Text, Any
from hasp.io import ContextBorg, get_key_list
from rich import print as rprint
from pprint import pprint
from copy import deepcopy


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
