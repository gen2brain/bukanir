package com.bukanir.android.entities;

public class AutoComplete {

    public String title;
    public String year;

    @Override
    public String toString() {
        StringBuilder result = new StringBuilder();
        String NL = System.getProperty("line.separator");

        result.append(((Object)this).getClass().getName() + " {" + NL);
        result.append("  title: " + this.title + NL);
        result.append("  year: " + this.year + NL);
        result.append("}" + NL);

        return result.toString();
    }

}
