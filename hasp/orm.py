from __future__ import annotations

from typing import Iterable

from sqlalchemy import Column, Integer, String, Table, ForeignKey, Sequence
from sqlalchemy.orm import relationship
from sqlalchemy.ext.declarative import declarative_base


Base = declarative_base()

keys__key_groups__table = Table(
    'keys__key_groups', Base.metadata,
    Column('ssh_key_pk', String, ForeignKey('ssh_keys.name')),
    Column('ssh_key_group_pk', Integer, ForeignKey('ssh_key_groups.name'))
)


# ----------------------------------------------------------------------------------------------------------------------
# Keys

class SshKey(Base):
    __tablename__ = 'ssh_keys'
    name = Column(String, primary_key=True)
    type = Column(String)
    comment = Column(String)
    bits = Column(Integer)
    md5 = Column(String)
    format = Column(String)

    ssh_key_groups = relationship(
        "SshKeyGroup",
        secondary=keys__key_groups__table,
        back_populates="ssh_keys"
    )
    ssh_key_aliases: Iterable[SshKeyAlias] = relationship("SshKeyAlias", back_populates="ssh_key")

    def __repr__(self):
        return self.name


class SshKeyGroup(Base):
    __tablename__ = 'ssh_key_groups'
    name = Column(String, primary_key=True)

    ssh_keys = relationship(
        "SshKey",
        secondary=keys__key_groups__table,
        back_populates="ssh_key_groups"
    )

    def __repr__(self):
        return self.name


class SshKeyAlias(Base):
    __tablename__ = 'ssh_key_aliases'
    name = Column(String, primary_key=True)
    ssh_key_pk = Column(String, ForeignKey('ssh_keys.name'))

    ssh_key = relationship("SshKey", back_populates="ssh_key_aliases")

    def __repr__(self):
        return self.name


# ----------------------------------------------------------------------------------------------------------------------
# Hosts

# class SshHost(Base):
#     __tablename__ = 'ssh_hosts'
#     id = Column(Integer, Sequence('user_id_seq'), primary_key=True)
#     name = Column(String)
#     ssh_keys = relationship(
#         "SshKey",
#         secondary=keys_hosts_table,
#         back_populates="ssh_hosts"
#     )
#
#
# class SshHostGroup(Base):
#     __tablename__ = 'ssh_host'
#     id = Column(Integer, Sequence('user_id_seq'), primary_key=True)
#     name = Column(String)
#     ssh_keys = relationship(
#         "SshKey",
#         secondary=keys_hosts_table,
#         back_populates="ssh_hosts"
#     )
#
#
# # ----------------------------------------------------------------------------------------------------------------------
# # Configs
#
# class SshConfig(Base):
#     __tablename__ = 'ssh_host'
#     id = Column(Integer, Sequence('user_id_seq'), primary_key=True)
#     name = Column(String)
#     ssh_keys = relationship(
#         "SshKey",
#         secondary=keys_hosts_table,
#         back_populates="ssh_hosts"
#     )
#
#
# class SshConfigGroup(Base):
#     __tablename__ = 'ssh_host'
#     id = Column(Integer, Sequence('user_id_seq'), primary_key=True)
#     name = Column(String)
#     ssh_keys = relationship(
#         "SshKey",
#         secondary=keys_hosts_table,
#         back_populates="ssh_hosts"
#     )
