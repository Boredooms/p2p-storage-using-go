// Prime number checker in C
// Compile: emcc prime_checker.c -o prime_checker.wasm -s STANDALONE_WASM -O3

#include <stdio.h>
#include <stdbool.h>
#include <math.h>

bool is_prime(int n) {
    if (n <= 1) return false;
    if (n <= 3) return true;
    if (n % 2 == 0 || n % 3 == 0) return false;
    
    for (int i = 5; i * i <= n; i += 6) {
        if (n % i == 0 || n % (i + 2) == 0)
            return false;
    }
    return true;
}

int count_primes(int limit) {
    int count = 0;
    for (int i = 2; i <= limit; i++) {
        if (is_prime(i)) {
            count++;
        }
    }
    return count;
}

int main() {
    int limit = 1000;
    
    printf("Checking primes up to %d...\n", limit);
    
    int prime_count = count_primes(limit);
    
    printf("Found %d prime numbers\n", prime_count);
    
    // Print first 10 primes
    printf("\nFirst 10 primes: ");
    int found = 0;
    for (int i = 2; found < 10; i++) {
        if (is_prime(i)) {
            printf("%d ", i);
            found++;
        }
    }
    printf("\n");
    
    return prime_count;
}
