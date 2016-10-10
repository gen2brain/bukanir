package com.bukanir.android.helpers;


import android.content.Context;
import android.content.DialogInterface;
import android.support.v7.app.AlertDialog;
import android.view.LayoutInflater;
import android.view.View;

import com.bukanir.android.R;

public class Dialogs {

    public static void showAbout(Context ctx) {
        LayoutInflater inflater = (LayoutInflater) ctx.getSystemService(Context.LAYOUT_INFLATER_SERVICE);
        View messageView = inflater.inflate(R.layout.dialog_about, null, false);

        String ver = Update.getCurrentVersion(ctx);
        String title = String.format("%s %s", ctx.getResources().getString(R.string.app_name), ver);

        AlertDialog.Builder builder = new AlertDialog.Builder(ctx);
        builder.setIcon(R.drawable.ic_launcher);
        builder.setTitle(title);
        builder.setView(messageView);
        builder.create();
        builder.show();
    }

    public static void showUpdate(Context ctx) {
        final Context context = ctx;
        AlertDialog.Builder builder = new AlertDialog.Builder(ctx);
        builder.setIcon(R.drawable.ic_launcher);
        builder.setTitle(R.string.update_available);
        builder.setMessage(R.string.update_download);
        builder.setPositiveButton("OK",
                new DialogInterface.OnClickListener() {
                    public void onClick(DialogInterface dialog, int whichButton) {
                        Update.downloadUpdate(context);
                    }
                }
        );
        builder.setNegativeButton("Cancel",
                new DialogInterface.OnClickListener() {
                    public void onClick(DialogInterface dialog, int whichButton) {
                        dialog.dismiss();
                    }
                }
        );
        builder.create();
        builder.show();
    }
}
