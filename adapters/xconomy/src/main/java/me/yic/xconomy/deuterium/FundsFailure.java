package me.yic.xconomy.deuterium;

/** GPL-3.0-or-later; part of the Deuterium XConomy extension. */
public final class FundsFailure extends RuntimeException {
    private final String code;
    public FundsFailure(String code,String message){super(message);this.code=code;}
    public FundsFailure(String code,String message,Throwable cause){super(message,cause);this.code=code;}
    public String code(){return code;}
}
