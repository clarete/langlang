// parsing should succeed
public class Example {
    public static void main(String[] args) {
        int n = 5;
        int f = 1;
        while(0 < n) {
            f = f * n;
            n = n - 1;
        }
        System.out.println(f);
    }
}
