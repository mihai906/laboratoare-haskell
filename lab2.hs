import Data.List


double :: Integer -> Integer
double x = x + x

maxim :: Integer -> Integer -> Integer
maxim x y = if (x > y) then x else y

-- varianta cu indentare

maxim2 :: Integer -> Integer -> Integer
maxim2 x y =
    if x > y
        then x
        else y

maxim3 :: Integer -> Integer -> Integer -> Integer
maxim3 x y z = maxim x (maxim y z)

-- maxim3 doar cu if si indentari

maxim3v2 :: Integer -> Integer -> Integer -> Integer
maxim3v2 x y z =
    if x >= y && x >= z
        then x
        else if y >= z
            then y
            else z

-- sau

maxim3v3 :: Integer -> Integer -> Integer -> Integer
maxim3v3 x y z =
    let
        u = maxim x y
    in
        maxim u z

maxim4 :: Integer -> Integer -> Integer -> Integer -> Integer
maxim4 w x y z =
    let
        u = maxim w x
        v = maxim y z
    in
        maxim u v

testMaxim4 :: Integer -> Integer -> Integer -> Integer -> Bool
testMaxim4 w x y z =
    let
        m = maxim4 w x y z
    in
        m >= w && m >= x && m >= y && m >= z


-- ex 6
-- a)
squareSum :: Integer -> Integer -> Integer
squareSum x y = x^2 + y^2

-- b)
parity :: Integer -> String
parity x = if ( x `mod` 2 == 0) then "par" else "impar"

-- c)
factorial :: Integer -> Integer
factorial 0 = 1
factorial x = x * factorial(x - 1)

-- d)
d :: Integer -> Integer -> Bool
d x y =
    if x > 2 * y
        then True
        else False

-- e)
maximLista :: [Integer] -> Integer
maximLista [x] = x
maximLista (x:xs) = maxim x (maximLista xs)

-- ex 7
poly :: Double -> Double -> Double -> Double -> Double
poly a b c x = a * x^2 + b * x + c

-- ex 8
fizzbuzz :: Integer -> String
fizzbuzz x =
    if x `mod` 15 == 0
        then "FizzBuzz"
        else if x `mod` 5 == 0
            then "Buzz"
            else if x `mod` 3 == 0
                then "Fizz"
                else ""

fizzbuzzv2 :: Integer -> String
fizzbuzzv2 x
    | x `mod` 15 == 0 =  "FizzBuzz"
    | x `mod` 3 == 0 = "Fizz"
    | x `mod` 5 == 0 = "Buzz"
    | otherwise = ""


fibonacciCazuri :: Integer -> Integer
fibonacciCazuri n
    | n < 2 = n
    | otherwise = fibonacciCazuri(n - 1) + fibonacciCazuri(n - 2)


fibonacciEcuational :: Integer -> Integer
fibonacciEcuational 0 = 0
fibonacciEcuational 1 = 1
fibonacciEcuational n =
    fibonacciEcuational(n - 1) + fibonacciEcuational(n - 2)


tribonacciCazuri :: Integer -> Integer
tribonacciCazuri n
    | n == 1 = 1
    | n == 2 = 1
    | n == 3 = 2
    | otherwise = tribonacciCazuri(n - 1) + tribonacciCazuri(n - 2) +
    tribonacciCazuri(n - 3)


tribonacciEcuational :: Integer -> Integer
tribonacciEcuational 1 = 1
tribonacciEcuational 2 = 1
tribonacciEcuational 3 = 2
tribonacciEcuational n =
    tribonacciEcuational(n - 1) + tribonacciEcuational(n - 2) +
    tribonacciEcuational(n - 3)


binomial :: Integer -> Integer -> Integer
binomial n k
    | k == 0 = 1
    | n == 0 = 0
    | otherwise = binomial (n - 1) k + binomial (n - 1) (k - 1)


-- exercitii suplimentare
-- ex 11
-- a)
sumN :: Integer -> Integer
sumN 1 = 1
sumN n = n + sumN(n - 1)

-- b)
putere :: Integer -> Integer -> Integer
putere x 0 = 1
putere x y = x * putere x (y - 1)

-- c)
inmultire :: Integer -> Integer -> Integer
inmultire _ 0 = 0
inmultire x y = x + inmultire x (y - 1)

-- d)
cifre :: Integer -> Integer
cifre 0 = 0
cifre x = (x `mod` 10) + cifre(x `div` 10)

-- exercitii bonus
-- a)
nrCifre :: Integer -> Integer
nrCifre x
    | x <= 9 = 1
    | otherwise = nrCifre(x `div` 10) + 1

-- b)
prodCifPare :: Integer -> Integer
prodCifPare 0 = 1
prodCifPare x
    | u `mod` 2 == 0 = u * prodCifPare rest
    | otherwise = prodCifPare rest
    where
    u = x `mod` 10
    rest = x `div` 10

-- c)
-- oglindit :: Integer -> Integer


-- d)
prim :: Integer -> Bool
prim n
    | n < 2 = False
    | otherwise = not(areDiv 2)
    where
    areDiv d
        | d * d > n = False
        | n `mod` d == 0 = True
        | otherwise = areDiv(d + 1)

-- e)
nrAparitii :: Integer -> Integer -> Integer
nrAparitii _ 0 = 0
nrAparitii c n
    | n `mod` 10 == c = 1 + nrAparitii c (n `div` 10)
    | otherwise = nrAparitii c (n `div` 10)


-- f)
stergeCifra :: Integer -> Integer -> Integer
stergeCifra _ 0 = 0
stergeCifra c n
    | u == c = stergeCifra c rest
    | otherwise = stergeCifra c rest * 10 + u
    where
    u = n `mod` 10
    rest = n `div` 10

-- g)
ackermann :: Integer -> Integer -> Integer
ackermann 0 n = n + 1
ackermann m 0 = ackermann(m - 1) 1
ackermann m n = ackermann(m - 1) (ackermann m (n - 1))
