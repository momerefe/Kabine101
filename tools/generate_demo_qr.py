"""Standart kitaplıkla tek, sabit demo QR varlığı üretir (QR v1-L, mask 0).

Uygulamanın çalışması için bu betiği çalıştırmak gerekmez.
QR içeriği yalnızca DEMO-KOCBENEFITS metnidir; erişim yetkisi sağlamaz.
"""
from pathlib import Path


def gf_multiply(a, b):
    result = 0
    while b:
        if b & 1:
            result ^= a
        a <<= 1
        if a & 0x100:
            a ^= 0x11D
        b >>= 1
    return result


def make_qr():
    payload = b"DEMO-KOCBENEFITS"
    bits = "0100" + f"{len(payload):08b}" + "".join(f"{byte:08b}" for byte in payload)
    bits += "0" * min(4, 152 - len(bits))
    bits += "0" * (-len(bits) % 8)
    data = [int(bits[i:i + 8], 2) for i in range(0, len(bits), 8)]
    pads = (0xEC, 0x11)
    while len(data) < 19:
        data.append(pads[(len(data) - len(bits) // 8) % 2])

    generator, root = [1], 1
    for _ in range(7):
        polynomial = [0] * (len(generator) + 1)
        for i, value in enumerate(generator):
            polynomial[i] ^= value
            polynomial[i + 1] ^= gf_multiply(value, root)
        generator = polynomial
        root = gf_multiply(root, 2)
    remainder = data + [0] * 7
    for i in range(19):
        value = remainder[i]
        for j, coefficient in enumerate(generator):
            remainder[i + j] ^= gf_multiply(value, coefficient)
    encoded = "".join(f"{value:08b}" for value in data + remainder[-7:])

    size = 21
    matrix = [[False] * size for _ in range(size)]
    reserved = [[False] * size for _ in range(size)]

    def put(x, y, value):
        if 0 <= x < size and 0 <= y < size:
            matrix[y][x] = bool(value)
            reserved[y][x] = True

    for origin_x, origin_y in ((0, 0), (14, 0), (0, 14)):
        for dy in range(-1, 8):
            for dx in range(-1, 8):
                dark = (0 <= dx <= 6 and 0 <= dy <= 6 and
                        (dx in (0, 6) or dy in (0, 6) or (2 <= dx <= 4 and 2 <= dy <= 4)))
                put(origin_x + dx, origin_y + dy, dark)
    for i in range(8, size - 8):
        put(6, i, i % 2 == 0)
        put(i, 6, i % 2 == 0)

    format_value = 8  # L error correction, mask 0
    remainder_value = format_value
    for _ in range(10):
        remainder_value = (remainder_value << 1) ^ ((remainder_value >> 9) * 0x537)
    format_bits = ((format_value << 10) | remainder_value) ^ 0x5412
    bit = lambda i: (format_bits >> i) & 1
    for i in range(6):
        put(8, i, bit(i))
    put(8, 7, bit(6))
    put(8, 8, bit(7))
    put(7, 8, bit(8))
    for i in range(9, 15):
        put(14 - i, 8, bit(i))
    for i in range(8):
        put(size - 1 - i, 8, bit(i))
    for i in range(8, 15):
        put(8, size - 15 + i, bit(i))
    put(8, size - 8, True)

    cursor = 0
    right = size - 1
    while right >= 1:
        if right == 6:
            right = 5
        upwards = (right + 1) & 2 == 0
        for vertical in range(size):
            y = size - 1 - vertical if upwards else vertical
            for offset in range(2):
                x = right - offset
                if not reserved[y][x]:
                    raw = int(encoded[cursor]) if cursor < len(encoded) else 0
                    matrix[y][x] = bool(raw ^ ((x + y) % 2 == 0))
                    cursor += 1
        right -= 2
    assert cursor == len(encoded) == 208
    path = "".join(f"M{x + 4} {y + 4}h1v1h-1z" for y in range(size) for x in range(size) if matrix[y][x])
    return f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 29 29" shape-rendering="crispEdges"><title>DEMO-KOCBENEFITS</title><path fill="white" d="M0 0h29v29H0z"/><path fill="#202020" d="{path}"/></svg>\n'


if __name__ == "__main__":
    target = Path(__file__).resolve().parent.parent / "assets" / "demo-qr.svg"
    target.write_text(make_qr(), encoding="utf-8")
    print("Demo QR asset generated.")
