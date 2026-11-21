#!/usr/bin/env python3
"""Embed existing multi-size Hyspayre .ico as real Windows PE icon resources.

Zero external dependencies. Writes a COFF .syso with RT_ICON and RT_GROUP_ICON,
including RVA relocations. Go's Windows linker includes this in EXE builds.
"""
from pathlib import Path
import struct

ROOT = Path(__file__).resolve().parents[1]
ICON = ROOT / 'setup' / 'hyspayre.ico'


def pad4(b):
    return b + b'\0' * ((-len(b)) % 4)


def parse_icon(b):
    if len(b) < 6:
        raise ValueError('Invalid .ico')
    zero, kind, count = struct.unpack_from('<HHH', b)
    if zero or kind != 1 or not (1 <= count <= 30):
        raise ValueError('Expected multi-resolution Windows .ico')
    entries = []
    for i in range(count):
        entry = b[6 + 16*i : 6 + 16*(i+1)]
        if len(entry) != 16:
            raise ValueError('Truncated .ico')
        width, height, colors, reserved, planes, bits, size, offset = struct.unpack('<BBBBHHII',entry)
        blob=b[offset:offset+size]
        if len(blob)!=size or size == 0:
            raise ValueError('Invalid icon image payload')
        entries.append((width,height,colors,reserved,planes,bits,blob))
    return entries


def make_rsrc(ico_bytes):
    entries = parse_icon(ico_bytes)
    # Each directory is a 16-byte header followed by 8-byte entries.
    blob = bytearray()
    def reserve(n):
        p = len(blob); blob.extend(b'\x00'*n); return p
    def make_dir(number):
        p=reserve(16+number*8)
        struct.pack_into('<HH',blob,p+12,0,number)
        return p
    def pointer(directory, index, resource_id, off, is_dir):
        struct.pack_into('<II',blob,directory+16+8*index,resource_id,off|(0x80000000 if is_dir else 0))

    root=make_dir(2)
    icon_type=make_dir(len(entries))
    group_type=make_dir(1)
    icon_langs=[make_dir(1) for _ in entries]
    group_lang=make_dir(1)
    icon_data=[reserve(16) for _ in entries]
    group_data=reserve(16)

    pointer(root,0,3,icon_type,True)
    pointer(root,1,14,group_type,True)
    pointer(group_type,0,1,group_lang,True)
    pointer(group_lang,0,1033,group_data,False)
    for i,lang in enumerate(icon_langs):
        pointer(icon_type,i,i+1,lang,True)
        pointer(lang,0,1033,icon_data[i],False)

    group=bytearray(struct.pack('<HHH',0,1,len(entries)))
    for i,(w,h,c,r,planes,bits,data) in enumerate(entries):
        group.extend(struct.pack('<BBBBHHIH',w,h,c,r,planes,bits,len(data),i+1))

    # Data RVA fields below are section-relative addends. Relocations make
    # them valid VA-independent RVAs when the PE image gets linked.
    relocs=[]
    for data_entry, data in zip(icon_data,[e[6] for e in entries]):
        blob.extend(b'\0'*((-len(blob))%4))
        data_off=len(blob)
        blob.extend(pad4(data))
        struct.pack_into('<IIII',blob,data_entry,data_off,len(data),0,0)
        relocs.append(data_entry)
    blob.extend(b'\0'*((-len(blob))%4))
    group_off=len(blob)
    blob.extend(pad4(group))
    struct.pack_into('<IIII',blob,group_data,group_off,len(group),0,0)
    relocs.append(group_data)
    return bytes(blob),relocs


def as_coff(resource, relocations):
    # Windows AMD64 COFF with a single .rsrc section, one section symbol,
    # and IMAGE_REL_AMD64_ADDR32NB relocations (type 0x0003).
    machine=0x8664
    ptr_raw=20+40
    ptr_reloc=ptr_raw+len(resource)
    num_reloc=len(relocations)
    ptr_symbols=ptr_reloc+10*num_reloc
    hdr=struct.pack('<HHIIIHH',machine,1,0,ptr_symbols,1,0,0)
    sec=struct.pack('<8sIIIIIIHHI',b'.rsrc\0\0\0',0,0,len(resource),ptr_raw,ptr_reloc,0,num_reloc,0,0x40000040)
    reloc_bytes=b''.join(struct.pack('<IIH',offset,0,0x0003) for offset in relocations)
    symbol=struct.pack('<8sIhHBB',b'.rsrc\0\0\0',0,1,0,3,0)
    return hdr+sec+resource+reloc_bytes+symbol+struct.pack('<I',4)


def main():
    resource,relocations=make_rsrc(ICON.read_bytes())
    data=as_coff(resource,relocations)
    for target in [ROOT/'cmd/launcher/rsrc_windows_amd64.syso',ROOT/'setup/rsrc_windows_amd64.syso']:
        target.write_bytes(data)
        print(f'Generated {target.relative_to(ROOT)} ({len(data)} bytes, {len(relocations)} icons)')

if __name__=='__main__':
    main()
