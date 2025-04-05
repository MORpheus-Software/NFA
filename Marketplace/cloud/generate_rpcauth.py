#!/usr/bin/env python3

import hmac
import secrets

def generate_salt(size):
    """Create size byte hex salt"""
    return secrets.token_hex(size)

def password_to_hmac(salt, password):
    """Create HMAC-SHA256 hash from password and salt"""
    m = hmac.new(salt.encode('utf-8'), password.encode('utf-8'), 'SHA256')
    return m.hexdigest()

def main():
    username = 'proxy'
    password = 'yosz9BZCuu7Rli7mYe4G1JbIO0Yprvwl'
    
    # Create 16 byte hex salt
    salt = generate_salt(16)
    password_hmac = password_to_hmac(salt, password)
    
    print('String to be appended to proxy.conf:')
    print(f'rpcauth={username}:{salt}${password_hmac}')
    print(f'Your password:\n{password}')

if __name__ == '__main__':
    main() 