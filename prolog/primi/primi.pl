% vero se N e' primo
is_prime(2).
is_prime(3).
is_prime(N) :-
    integer(N),
    N > 3,
    N mod 2 =\= 0,
    \+ has_factor(N, 3).

has_factor(N, F) :-
    F * F =< N,
    (   N mod F =:= 0
    ;   F2 is F + 2,
        has_factor(N, F2)
    ).

% conta i primi nell'intervallo [A, B]
count_primes(A, B, Count) :-
    findall(N, (between(A, B, N), is_prime(N)), Primes),
    length(Primes, Count).

% stampa il numero di primi ogni 100 naturali, fino a Max
primi_ogni_100(Max) :-
    Max >= 1,
    primi_ogni_100_da(1, Max, 0, Totale),
    format('Totale primi trovati: ~d~n', [Totale]).

primi_ogni_100_da(Start, Max, Acc, Totale) :-
    Start =< Max,
    End is min(Start + 99, Max),
    count_primes(Start, End, Count),
    format('~d-~d: ~d~n', [Start, End, Count]),
    Acc1 is Acc + Count,
    Next is Start + 100,
    primi_ogni_100_da(Next, Max, Acc1, Totale).
primi_ogni_100_da(Start, Max, Totale, Totale) :-
    Start > Max.
